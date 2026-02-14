package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeJSON(t *testing.T, path string, v map[string]any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRun_MergesNewKeys(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	writeJSON(t, masterPath, map[string]any{"fromMaster": "yes", "shared": "master"})
	writeJSON(t, localPath, map[string]any{"fromLocal": "yes", "shared": "local"})

	var buf bytes.Buffer
	if err := run(masterPath, localPath, false, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := readJSON(t, localPath)

	if result["fromMaster"] != "yes" {
		t.Errorf("fromMaster = %v; want yes", result["fromMaster"])
	}
	if result["fromLocal"] != "yes" {
		t.Errorf("fromLocal = %v; want yes", result["fromLocal"])
	}
	if result["shared"] != "local" {
		t.Errorf("shared = %v; want local (local value must win)", result["shared"])
	}
}

func TestRun_ReportsConflictsInOutput(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	writeJSON(t, masterPath, map[string]any{"key": "master-value"})
	writeJSON(t, localPath, map[string]any{"key": "local-value"})

	var buf bytes.Buffer
	if err := run(masterPath, localPath, false, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "Conflicts") {
		t.Errorf("expected Conflicts in output, got:\n%s", buf.String())
	}
}

func TestRun_MatchingKeysSkipsWrite(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	writeJSON(t, masterPath, map[string]any{"key": "same"})
	writeJSON(t, localPath, map[string]any{"key": "same"})

	var buf bytes.Buffer
	if err := run(masterPath, localPath, false, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "No changes to write") {
		t.Errorf("expected 'No changes to write' in output, got:\n%s", buf.String())
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".bak") {
			t.Errorf("expected no backup file, found %s", e.Name())
		}
	}
}

func TestRun_CreatesBackupFile(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	writeJSON(t, masterPath, map[string]any{"key": "value"})
	writeJSON(t, localPath, map[string]any{})

	var buf bytes.Buffer
	if err := run(masterPath, localPath, false, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	var hasBackup bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "local.json.") && strings.HasSuffix(e.Name(), ".bak") {
			hasBackup = true
		}
	}
	if !hasBackup {
		t.Error("expected a backup file to be created")
	}
}

func TestRun_MissingMaster(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "local.json")
	writeJSON(t, localPath, map[string]any{})

	err := run("/nonexistent/master.json", localPath, false, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for missing master, got nil")
	}
}

func TestRun_MissingLocal(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	writeJSON(t, masterPath, map[string]any{})

	err := run(masterPath, "/nonexistent/local.json", false, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for missing local, got nil")
	}
}

func TestLoadJSON_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`not json`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := loadJSON(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestRun_WriteFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test: running as root")
	}

	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	writeJSON(t, masterPath, map[string]any{"key": "value"})
	writeJSON(t, localPath, map[string]any{})

	// Make the directory non-writable so os.CreateTemp (and any rename) fails.
	if err := os.Chmod(dir, 0o555); err != nil { //nolint:gosec // 0o555 is intentional to test write failure
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) //nolint:gosec // restoring directory to normal permissions after test

	err := run(masterPath, localPath, false, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error when writing to read-only directory, got nil")
	}
}

func TestRun_BackupFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test: running as root")
	}

	// Use a subdirectory so we can make localPath's directory read-only
	// without affecting masterPath.
	baseDir := t.TempDir()
	masterPath := filepath.Join(baseDir, "master.json")
	localDir := filepath.Join(baseDir, "local")
	if err := os.MkdirAll(localDir, 0o750); err != nil {
		t.Fatal(err)
	}
	localPath := filepath.Join(localDir, "settings.json")

	writeJSON(t, masterPath, map[string]any{"key": "value"})
	writeJSON(t, localPath, map[string]any{})

	// Make the local directory read-only so backup.Create fails when it
	// tries to write the backup file there.
	if err := os.Chmod(localDir, 0o555); err != nil { //nolint:gosec // 0o555 intentional to test backup failure
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(localDir, 0o755) }) //nolint:gosec // restore directory permissions after test

	var buf bytes.Buffer
	err := run(masterPath, localPath, false, &buf)
	if err == nil {
		t.Fatal("expected error when backup directory is read-only, got nil")
	}
}

func TestRun_ListsAddedKeys(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	writeJSON(t, masterPath, map[string]any{"newKey": "from-master", "anotherNew": "also-from-master"})
	writeJSON(t, localPath, map[string]any{})

	var buf bytes.Buffer
	if err := run(masterPath, localPath, false, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Keys added:") {
		t.Errorf("expected 'Keys added:' section in output, got:\n%s", output)
	}
	if !strings.Contains(output, "newKey") {
		t.Errorf("expected 'newKey' listed in output, got:\n%s", output)
	}
	if !strings.Contains(output, "anotherNew") {
		t.Errorf("expected 'anotherNew' listed in output, got:\n%s", output)
	}
}

func TestRun_ListsMatchingKeys(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	writeJSON(t, masterPath, map[string]any{"sharedKey": "same-value", "newKey": "from-master"})
	writeJSON(t, localPath, map[string]any{"sharedKey": "same-value"})

	var buf bytes.Buffer
	if err := run(masterPath, localPath, false, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Matching keys:") {
		t.Errorf("expected 'Matching keys:' section in output, got:\n%s", output)
	}
	if !strings.Contains(output, "sharedKey") {
		t.Errorf("expected 'sharedKey' listed under matching keys, got:\n%s", output)
	}
}

func TestRun_ConflictWithNestedObjectValue(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	writeJSON(t, masterPath, map[string]any{"key": map[string]any{"nested": "master-val"}})
	writeJSON(t, localPath, map[string]any{"key": map[string]any{"nested": "local-val"}})

	var buf bytes.Buffer
	if err := run(masterPath, localPath, false, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Conflicts") {
		t.Errorf("expected 'Conflicts' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "master:") {
		t.Errorf("expected 'master:' label in output, got:\n%s", output)
	}
	if !strings.Contains(output, "local:") {
		t.Errorf("expected 'local:' label in output, got:\n%s", output)
	}
}

func TestRun_ConflictWithSliceValue(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	if err := os.WriteFile(masterPath, []byte(`{"key":["a","b"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localPath, []byte(`{"key":["c","d"]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := run(masterPath, localPath, false, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Conflicts") {
		t.Errorf("expected 'Conflicts' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "master:") {
		t.Errorf("expected 'master:' label in output, got:\n%s", output)
	}
	if !strings.Contains(output, "local:") {
		t.Errorf("expected 'local:' label in output, got:\n%s", output)
	}
}

func TestRun_ListsLocalOnlyKeys(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	writeJSON(t, masterPath, map[string]any{"masterKey": "value"})
	writeJSON(t, localPath, map[string]any{"localOnlyKey": "local-value", "masterKey": "value"})

	var buf bytes.Buffer
	if err := run(masterPath, localPath, false, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Local-only keys (not in master):") {
		t.Errorf("expected 'Local-only keys (not in master):' section in output, got:\n%s", output)
	}
	if !strings.Contains(output, "localOnlyKey") {
		t.Errorf("expected 'localOnlyKey' listed under local-only keys, got:\n%s", output)
	}
}

func TestRun_ForceOnlyNoEmptyAddedSection(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	// Master and local share one key with different values; no new keys from master.
	writeJSON(t, masterPath, map[string]any{"key": "master-value"})
	writeJSON(t, localPath, map[string]any{"key": "local-value"})

	var buf bytes.Buffer
	if err := run(masterPath, localPath, true, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// "Keys added:\n" is the section header (followed immediately by a newline
	// and key listings). The summary line prints "Keys added: 0" (with a number),
	// so checking for the header form ensures the section block is absent.
	if strings.Contains(output, "Keys added:\n") {
		t.Errorf("expected no 'Keys added:' section when nothing was added, got:\n%s", output)
	}
	if !strings.Contains(output, "Forced overwrites") {
		t.Errorf("expected 'Forced overwrites' in output, got:\n%s", output)
	}
}

func TestRun_ForceOverwritesConflict(t *testing.T) {
	dir := t.TempDir()
	masterPath := filepath.Join(dir, "master.json")
	localPath := filepath.Join(dir, "local.json")

	writeJSON(t, masterPath, map[string]any{"key": "master-value"})
	writeJSON(t, localPath, map[string]any{"key": "local-value"})

	var buf bytes.Buffer
	if err := run(masterPath, localPath, true, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := readJSON(t, localPath)
	if result["key"] != "master-value" {
		t.Errorf("key = %v; want master-value (master must win with force)", result["key"])
	}

	output := buf.String()
	if !strings.Contains(output, "Forced overwrites") {
		t.Errorf("expected 'Forced overwrites' in output, got:\n%s", output)
	}
}
