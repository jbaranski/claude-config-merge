// Package dirsync copies files and directories from one directory to another.
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
	Copied  []string // entries copied (new)
	Skipped []string // entries skipped (already exist, no force)
	Forced  []string // entries overwritten because force=true
}

// Sync copies regular files and subdirectories from src to dst.
// If force is false, existing entries in dst are skipped.
// If force is true, existing entries in dst are overwritten.
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

	if err := os.MkdirAll(dst, 0o750); err != nil && !errors.Is(err, os.ErrExist) {
		return res, fmt.Errorf("creating destination directory %s: %w", dst, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		srcPath := filepath.Join(src, name)
		dstPath := filepath.Join(dst, name)

		// Use Lstat so broken/circular symlinks are treated as "exists"
		// rather than causing an infinite-follow error.
		_, statErr := os.Lstat(dstPath)
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return res, fmt.Errorf("stat %s: %w", dstPath, statErr)
		}
		exists := statErr == nil

		if exists && !force {
			res.Skipped = append(res.Skipped, name)
			continue
		}

		switch {
		case entry.Type().IsRegular():
			if err := copyFile(srcPath, dstPath); err != nil {
				return res, err
			}
		case entry.IsDir():
			if err := copyDir(srcPath, dstPath); err != nil {
				return res, err
			}
		default:
			// Skip symlinks and other special types.
			continue
		}

		if exists {
			res.Forced = append(res.Forced, name)
		} else {
			res.Copied = append(res.Copied, name)
		}
	}

	return res, nil
}

// copyDir recursively copies the directory tree at src to dst.
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}

	if err := os.MkdirAll(dst, 0o750); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("creating %s: %w", dst, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		switch {
		case entry.IsDir():
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		case entry.Type().IsRegular():
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
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
