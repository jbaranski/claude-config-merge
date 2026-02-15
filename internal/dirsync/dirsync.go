// Package dirsync copies files and directories from one directory to another.
package dirsync

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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

	sort.Strings(res.Copied)
	sort.Strings(res.Skipped)
	sort.Strings(res.Forced)

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
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close() //nolint:errcheck // best-effort close of read-only source

	info, err := in.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".copy-*")
	if err != nil {
		return fmt.Errorf("creating temp file for %s: %w", dst, err)
	}
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(info.Mode()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("setting permissions on temp file: %w", err)
	}

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file for %s: %w", dst, err)
	}

	if err := os.Rename(tmpName, dst); err != nil {
		return fmt.Errorf("renaming temp to %s: %w", dst, err)
	}
	tmpName = "" // disarm the defer
	return nil
}
