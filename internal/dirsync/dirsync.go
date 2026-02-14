// Package dirsync copies files from one directory to another.
package dirsync

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Result holds the outcome of a directory sync operation.
type Result struct {
	Copied  []string // files copied (new)
	Skipped []string // files skipped (already exist, no force)
	Forced  []string // files overwritten because force=true
}

// Sync copies regular files from src to dst.
// If force is false, existing files in dst are skipped.
// If force is true, existing files in dst are overwritten.
// src not existing is not an error — returns empty Result.
// dst is created if it does not exist.
func Sync(src, dst string, force bool) (Result, error) {
	var res Result

	entries, err := os.ReadDir(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return res, nil
		}
		return res, fmt.Errorf("reading source directory %s: %w", src, err)
	}

	if err := os.MkdirAll(dst, 0o750); err != nil {
		return res, fmt.Errorf("creating destination directory %s: %w", dst, err)
	}

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		name := entry.Name()
		srcPath := filepath.Join(src, name)
		dstPath := filepath.Join(dst, name)

		_, statErr := os.Stat(dstPath)
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return res, fmt.Errorf("stat %s: %w", dstPath, statErr)
		}
		exists := statErr == nil

		if exists && !force {
			res.Skipped = append(res.Skipped, name)
			continue
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			return res, err
		}

		if exists {
			res.Forced = append(res.Forced, name)
		} else {
			res.Copied = append(res.Copied, name)
		}
	}

	return res, nil
}

// copyFile copies the file at src to dst, preserving the source file's permissions.
func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close() //nolint:errcheck // best-effort close of read-only source

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", dst, err)
	}

	return nil
}
