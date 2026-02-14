package backup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreate_WritesBackupFile(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "settings.json")
	content := []byte(`{"key": "value"}`)

	if err := os.WriteFile(original, content, 0o600); err != nil {
		t.Fatal(err)
	}

	backupPath, err := Create(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(backupPath, original+".") || !strings.HasSuffix(backupPath, ".bak") {
		t.Errorf("backupPath = %q; want format %q", backupPath, original+".TIMESTAMP.bak")
	}

	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Errorf("backup content = %q; want %q", string(got), string(content))
	}
}

func TestCreate_PreservesOriginalFile(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "settings.json")
	content := []byte(`{"key": "value"}`)

	if err := os.WriteFile(original, content, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Create(original); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(original)
	if err != nil {
		t.Fatalf("reading original: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Errorf("original content changed; got %q, want %q", string(got), string(content))
	}
}

func TestCreate_MissingFile(t *testing.T) {
	_, err := Create("/nonexistent/settings.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestCreate_WriteFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test: running as root")
	}

	dir := t.TempDir()
	original := filepath.Join(dir, "settings.json")

	if err := os.WriteFile(original, []byte(`{"key": "value"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Make the directory read-only so os.WriteFile for the backup will fail.
	if err := os.Chmod(dir, 0o555); err != nil { //nolint:gosec // 0o555 intentional to test write failure
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) //nolint:gosec // restore directory permissions after test

	_, err := Create(original)
	if err == nil {
		t.Fatal("expected error when backup directory is read-only, got nil")
	}
}
