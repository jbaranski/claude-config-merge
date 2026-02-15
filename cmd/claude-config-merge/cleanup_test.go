package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCleanupBak_DeletesBackups(t *testing.T) {
	dir := t.TempDir()

	bakFiles := []string{
		"settings.json.20240101T120000.bak",
		"settings.json.20240102T130000.bak",
	}
	nonBakFile := "settings.json"

	for _, name := range bakFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("backup"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, nonBakFile), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runCleanupBak(dir, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Deleted") {
		t.Errorf("expected 'Deleted' in output, got:\n%s", output)
	}

	// Bak files must be gone.
	for _, name := range bakFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected %s to be deleted, but it still exists", name)
		}
	}

	// Non-bak file must remain.
	if _, err := os.Stat(filepath.Join(dir, nonBakFile)); err != nil {
		t.Errorf("expected %s to remain, got error: %v", nonBakFile, err)
	}
}

func TestRunCleanupBak_NoBackupsFound(t *testing.T) {
	dir := t.TempDir()

	var buf bytes.Buffer
	if err := runCleanupBak(dir, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No backup files found") {
		t.Errorf("expected 'No backup files found' in output, got:\n%s", output)
	}
}

func TestRunCleanupBak_MissingDir(t *testing.T) {
	var buf bytes.Buffer
	err := runCleanupBak("/nonexistent/path/that/does/not/exist", &buf)
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

func TestRunCleanupBak_PartialFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test: running as root")
	}

	dir := t.TempDir()

	bakFiles := []string{
		"settings.json.20240101T120000.bak",
		"settings.json.20240102T130000.bak",
	}
	for _, name := range bakFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("backup"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// Make the directory read-only so that os.Remove will fail for all files.
	if err := os.Chmod(dir, 0o555); err != nil { //nolint:gosec // 0o555 is intentional to test deletion failure
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) //nolint:gosec // restore directory permissions after test

	var buf bytes.Buffer
	err := runCleanupBak(dir, &buf)

	output := buf.String()
	if err == nil {
		t.Fatal("expected error when deletion fails, got nil")
	}
	if !strings.Contains(output, "Error deleting") {
		t.Errorf("expected 'Error deleting' in output, got:\n%s", output)
	}
	// The summary line must always be printed; with 0 successes and 2 attempts.
	if !strings.Contains(output, "Deleted 0 of 2") {
		t.Errorf("expected 'Deleted 0 of 2' in output, got:\n%s", output)
	}
}

func TestRunCleanupBak_NonBakFilesUntouched(t *testing.T) {
	dir := t.TempDir()

	nonBakFiles := []string{
		"settings.json",
		"settings.json.bak.old",
		"other.txt",
	}
	for _, name := range nonBakFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var buf bytes.Buffer
	if err := runCleanupBak(dir, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No backup files found") {
		t.Errorf("expected 'No backup files found' in output, got:\n%s", output)
	}

	// All non-bak files must remain.
	for _, name := range nonBakFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s to remain, got error: %v", name, err)
		}
	}
}
