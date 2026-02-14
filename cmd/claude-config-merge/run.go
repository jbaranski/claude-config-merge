package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jeff/claude-config-merge/internal/backup"
	"github.com/jeff/claude-config-merge/internal/merge"
)

// run performs the merge of masterPath into localPath, writing output to w.
// When force is true, conflicting keys use the master value instead of keeping local.
// Returns an error if any step fails.
func run(masterPath, localPath string, force bool, w io.Writer) error {
	masterData, err := loadJSON(masterPath)
	if err != nil {
		return fmt.Errorf("failed to load master settings: %w", err)
	}

	localData, err := loadJSON(localPath)
	if err != nil {
		return fmt.Errorf("failed to load local settings (%s): %w", localPath, err)
	}

	result := merge.Merge(masterData, localData, force)

	printMergeReport(&result, w)

	if len(result.Added) == 0 && len(result.Forced) == 0 {
		fmt.Fprintf(w, "No changes to write. Keys added: 0  |  Conflicts: %d  |  Matching: %d  |  Local-only: %d\n", len(result.Conflicts), len(result.Matching), len(result.LocalOnly))
		return nil
	}

	backupPath, err := backup.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	fmt.Fprintf(w, "Backup created: %s\n", backupPath)

	out, err := json.MarshalIndent(result.Merged, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal merged settings: %w", err)
	}

	dir := filepath.Dir(localPath)
	tmp, err := os.CreateTemp(dir, ".settings-merge-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	// ensure temp is cleaned up on any error path
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(append(out, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, localPath); err != nil {
		return fmt.Errorf("failed to write merged settings: %w", err)
	}
	tmpName = "" // disarm the defer

	if len(result.Added) > 0 {
		fmt.Fprintf(w, "Keys added:\n")
		for _, k := range result.Added {
			fmt.Fprintf(w, "  %s\n", k)
		}
		fmt.Fprintf(w, "\n")
	}

	fmt.Fprintf(w, "Done. Keys added: %d  |  Forced: %d  |  Conflicts: %d  |  Matching: %d  |  Local-only: %d\n", len(result.Added), len(result.Forced), len(result.Conflicts), len(result.Matching), len(result.LocalOnly))
	fmt.Fprintf(w, "Written to: %s\n", localPath)

	return nil
}

// printMergeReport writes the conflict, forced, matching, and local-only
// sections of the merge report to w.
func printMergeReport(result *merge.Result, w io.Writer) {
	const sep = "  ------------------------------------------------------------"

	if len(result.Conflicts) > 0 {
		fmt.Fprintf(w, "\nConflicts (local value kept):\n")
		for _, c := range result.Conflicts {
			fmt.Fprintf(w, "\n%s\n", sep)
			fmt.Fprintf(w, "  %s\n", c.Key)
			fmt.Fprintf(w, "    master: %s\n", formatValue(c.MasterValue))
			fmt.Fprintf(w, "    local:  %s\n", formatValue(c.LocalValue))
		}
		fmt.Fprintf(w, "\n%s\n\n", sep)
	}

	if len(result.Forced) > 0 {
		fmt.Fprintf(w, "\nForced overwrites (master value applied):\n")
		for _, k := range result.Forced {
			fmt.Fprintf(w, "\n%s\n", sep)
			fmt.Fprintf(w, "  %s\n", k)
		}
		fmt.Fprintf(w, "\n%s\n\n", sep)
	}

	if len(result.Matching) > 0 {
		fmt.Fprintf(w, "Matching keys:\n")
		for _, k := range result.Matching {
			fmt.Fprintf(w, "  %s\n", k)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(result.LocalOnly) > 0 {
		fmt.Fprintf(w, "Local-only keys (not in master):\n")
		for _, k := range result.LocalOnly {
			fmt.Fprintf(w, "  %s\n", k)
		}
		fmt.Fprintf(w, "\n")
	}
}

// formatValue returns a human-readable string for a conflict value.
// Complex types (objects, arrays) are pretty-printed as indented JSON with
// a four-space prefix so they align under the "master:"/"local:" label in
// the conflict report. Scalars are rendered as compact JSON.
func formatValue(v any) string {
	switch v.(type) {
	case map[string]any, []any:
		b, err := json.MarshalIndent(v, "    ", "  ")
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// loadJSON reads a JSON file at path and unmarshals it into a map. It returns
// an informative error that includes the underlying parse error and reminds the
// caller that JSON does not support // or /* */ comments.
func loadJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w (note: if the file contains // or /* */ comments, remove them — they are not valid JSON)", path, err)
	}

	return m, nil
}
