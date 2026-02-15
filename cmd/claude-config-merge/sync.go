package main

import (
	"fmt"
	"io"
	"os"

	"github.com/jeff/claude-config-merge/internal/dirsync"
)

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// runSync syncs files from srcDir to dstDir, printing a report to w.
// label is the human-readable name used in output (e.g., "Agents").
// If srcDir does not exist, a short notice is printed and nil is returned.
// If dstDir is a symlink it is skipped with a warning — the tool will not
// follow or overwrite a symlink that may be managed by another process.
func runSync(srcDir, dstDir string, force bool, label string, w io.Writer) error {
	if info, err := os.Lstat(dstDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		fmt.Fprintf(w, "%s: destination %s is a symbolic link — skipping.\n", label, dstDir)
		fmt.Fprintf(w, "  If this symlink was created by mistake, remove it first: rm %q\n", dstDir)
		return nil
	}

	res, err := dirsync.Sync(srcDir, dstDir, force)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}

	total := len(res.Copied) + len(res.Skipped) + len(res.Forced)

	// Distinguish between "src did not exist" and "src existed but was empty".
	// dirsync.Sync returns an empty result for both cases, so we check directly.
	if total == 0 {
		// Re-stat to tell the two zero cases apart.
		if !dirExists(srcDir) {
			fmt.Fprintf(w, "%s: source directory not found, skipping (%s)\n", label, srcDir)
			return nil
		}
		fmt.Fprintf(w, "%s: nothing to sync\n", label)
		return nil
	}

	fmt.Fprintf(w, "%s: copied %d, skipped %d, forced %d\n", label, len(res.Copied), len(res.Skipped), len(res.Forced))

	if len(res.Copied) > 0 {
		fmt.Fprintf(w, "  Copied:\n")
		for _, name := range res.Copied {
			fmt.Fprintf(w, "    %s\n", name)
		}
	}

	if len(res.Skipped) > 0 {
		fmt.Fprintf(w, "  Skipped (use -f to overwrite):\n")
		for _, name := range res.Skipped {
			fmt.Fprintf(w, "    %s\n", name)
		}
	}

	if len(res.Forced) > 0 {
		fmt.Fprintf(w, "  Forced:\n")
		for _, name := range res.Forced {
			fmt.Fprintf(w, "    %s\n", name)
		}
	}

	return nil
}
