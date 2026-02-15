// Package backup creates timestamped copies of files before modification.
package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Create copies the file at path to a timestamped backup and returns the backup path.
func Create(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	backupPath := fmt.Sprintf("%s.%s.bak", path, time.Now().Format("20060102T150405.000"))

	dir := filepath.Dir(backupPath)
	tmp, err := os.CreateTemp(dir, ".backup-*")
	if err != nil {
		return "", fmt.Errorf("creating temp backup file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("writing backup: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("closing temp backup: %w", err)
	}
	if err := os.Rename(tmpName, backupPath); err != nil {
		return "", fmt.Errorf("finalising backup %s: %w", backupPath, err)
	}
	tmpName = "" // disarm defer
	return backupPath, nil
}
