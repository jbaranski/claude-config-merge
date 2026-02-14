// Package backup creates timestamped copies of files before modification.
package backup

import (
	"fmt"
	"os"
	"time"
)

// Create copies the file at path to a timestamped backup and returns the backup path.
func Create(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	backupPath := fmt.Sprintf("%s.%s.bak", path, time.Now().Format("20060102T150405.000"))

	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return "", fmt.Errorf("writing backup %s: %w", backupPath, err)
	}

	return backupPath, nil
}
