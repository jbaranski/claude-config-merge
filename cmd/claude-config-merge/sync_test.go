package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupSyncDirs creates a temp directory with src and dst subdirectories and
// places a single file in each (with different content). It returns src and dst.
func setupSyncDirs(t *testing.T, fileName, srcContent, dstContent string) (src, dst string) {
	t.Helper()
	dir := t.TempDir()
	src = filepath.Join(dir, "src")
	dst = filepath.Join(dir, "dst")
	if err := os.MkdirAll(src, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, fileName), []byte(srcContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if dstContent != "" {
		if err := os.WriteFile(filepath.Join(dst, fileName), []byte(dstContent), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return src, dst
}

func TestRunSync_CopiesAndReports(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	if err := os.MkdirAll(src, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"agent-a.md", "agent-b.md"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte(name+" content"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var buf bytes.Buffer
	if err := runSync(src, dst, false, "Agents", &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Copied:") {
		t.Errorf("expected 'Copied:' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "agent-a.md") {
		t.Errorf("expected 'agent-a.md' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "agent-b.md") {
		t.Errorf("expected 'agent-b.md' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Agents: copied 2") {
		t.Errorf("expected 'Agents: copied 2' in output, got:\n%s", output)
	}
}

// test opposite behaviors (skip vs overwrite) that require structurally similar setup.
//
//nolint:dupl // TestRunSync_SkipsExistingAndReports and TestRunSync_ForceOverwrites
func TestRunSync_SkipsExistingAndReports(t *testing.T) {
	src, dst := setupSyncDirs(t, "existing.md", "new content", "original content")

	var buf bytes.Buffer
	if err := runSync(src, dst, false, "Agents", &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Skipped") {
		t.Errorf("expected 'Skipped' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "existing.md") {
		t.Errorf("expected 'existing.md' in output, got:\n%s", output)
	}
	got, err := os.ReadFile(filepath.Join(dst, "existing.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original content" {
		t.Errorf("dst file content = %q; want original content", got)
	}
}

// test opposite behaviors (overwrite vs skip) that require structurally similar setup.
//
//nolint:dupl // TestRunSync_ForceOverwrites and TestRunSync_SkipsExistingAndReports
func TestRunSync_ForceOverwrites(t *testing.T) {
	src, dst := setupSyncDirs(t, "file.md", "new content", "old content")

	var buf bytes.Buffer
	if err := runSync(src, dst, true, "Skills", &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Forced:") {
		t.Errorf("expected 'Forced:' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "file.md") {
		t.Errorf("expected 'file.md' in output, got:\n%s", output)
	}
	got, err := os.ReadFile(filepath.Join(dst, "file.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new content" {
		t.Errorf("dst file content = %q; want new content", got)
	}
}

func TestRunSync_SourceNotExist(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "nonexistent")
	dst := filepath.Join(dir, "dst")

	var buf bytes.Buffer
	if err := runSync(src, dst, false, "Agents", &buf); err != nil {
		t.Fatalf("expected nil error for missing src, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "source directory not found") {
		t.Errorf("expected 'source directory not found' in output, got:\n%s", output)
	}
	if !strings.Contains(output, src) {
		t.Errorf("expected src path in output, got:\n%s", output)
	}
}

func TestRunSync_NothingToSync(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	// src exists but is empty — nothing to copy.
	if err := os.MkdirAll(src, 0o750); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := runSync(src, dst, false, "Skills", &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "nothing to sync") {
		t.Errorf("expected 'nothing to sync' in output, got:\n%s", output)
	}
}

func TestRunSync_SymlinkDstSkips(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	if err := os.MkdirAll(srcDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "file.md"), []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create a dangling symlink at the dst path.
	if err := os.Symlink("nonexistent", dst); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runSync(srcDir, dst, false, "Test", &buf)
	if err != nil {
		t.Fatalf("expected nil error for symlink dst, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "symbolic link") {
		t.Errorf("expected 'symbolic link' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "rm") {
		t.Errorf("expected 'rm' hint in output, got:\n%s", output)
	}
}
