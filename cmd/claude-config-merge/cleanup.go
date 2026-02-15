package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// runCleanupBak finds and deletes settings.json.*.bak files in claudeDir.
func runCleanupBak(claudeDir string, w io.Writer) error {
	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", claudeDir, err)
	}

	var bakFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "settings.json.") && strings.HasSuffix(name, ".bak") {
			bakFiles = append(bakFiles, name)
		}
	}

	if len(bakFiles) == 0 {
		fmt.Fprintf(w, "No backup files found in %s\n", claudeDir)
		return nil
	}

	var errs []error
	deleted := 0
	for _, name := range bakFiles {
		path := filepath.Join(claudeDir, name)
		if err := os.Remove(path); err != nil {
			fmt.Fprintf(w, "  Error deleting %s: %v\n", name, err)
			errs = append(errs, err)
			continue
		}
		fmt.Fprintf(w, "  Deleted: %s\n", name)
		deleted++
	}

	fmt.Fprintf(w, "\nDeleted %d of %d backup file(s) from %s\n", deleted, len(bakFiles), claudeDir)
	return errors.Join(errs...)
}
