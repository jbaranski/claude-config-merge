package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// configDir must point to an existing directory.
	cfg := map[string]string{"configDir": dir}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ConfigDir != dir {
		t.Errorf("ConfigDir = %q; want %q", got.ConfigDir, dir)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte(`not json`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoad_MissingConfigDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing configDir, got nil")
	}
}

func TestLoad_ConfigDirNotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	nonExistentDir := filepath.Join(dir, "does-not-exist")
	cfg := map[string]string{"configDir": nonExistentDir}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = Load(path)
	if err == nil {
		t.Fatal("expected error for non-existent configDir, got nil")
	}
}

func TestDefaultPath_ReturnsHomeRelativePath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Fatal("expected non-empty default path")
	}
	if filepath.Base(path) != ".claude-config-merge.json" {
		t.Errorf("DefaultPath base = %q; want .claude-config-merge.json", filepath.Base(path))
	}
}
