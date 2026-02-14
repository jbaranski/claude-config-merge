package dirsync_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeff/claude-config-merge/internal/dirsync"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// makeSrcDst creates a temp dir with src and dst subdirectories and returns
// their paths.
func makeSrcDst(t *testing.T) (src, dst string) {
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
	return src, dst
}

func TestSync_CopiesNewFiles(t *testing.T) {
	src, dst := makeSrcDst(t)

	writeFile(t, filepath.Join(src, "a.md"), "agent a")
	writeFile(t, filepath.Join(src, "b.md"), "agent b")

	res, err := dirsync.Sync(src, dst, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Copied) != 2 {
		t.Errorf("Copied = %v; want 2 entries", res.Copied)
	}
	if len(res.Skipped) != 0 {
		t.Errorf("Skipped = %v; want empty", res.Skipped)
	}
	if len(res.Forced) != 0 {
		t.Errorf("Forced = %v; want empty", res.Forced)
	}

	if readFile(t, filepath.Join(dst, "a.md")) != "agent a" {
		t.Error("a.md content mismatch in dst")
	}
	if readFile(t, filepath.Join(dst, "b.md")) != "agent b" {
		t.Error("b.md content mismatch in dst")
	}
}

// test opposite behaviors (skip vs overwrite) and necessarily share similar structure.
//
//nolint:dupl // TestSync_SkipsExistingWithoutForce and TestSync_ForcesOverwriteWithForce
func TestSync_SkipsExistingWithoutForce(t *testing.T) {
	src, dst := makeSrcDst(t)

	writeFile(t, filepath.Join(src, "a.md"), "new content")
	writeFile(t, filepath.Join(dst, "a.md"), "original content")

	res, err := dirsync.Sync(src, dst, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Skipped) != 1 || res.Skipped[0] != "a.md" {
		t.Errorf("Skipped = %v; want [a.md]", res.Skipped)
	}
	if len(res.Copied) != 0 {
		t.Errorf("Copied = %v; want empty", res.Copied)
	}
	if len(res.Forced) != 0 {
		t.Errorf("Forced = %v; want empty", res.Forced)
	}
	// Verify original content is preserved.
	if readFile(t, filepath.Join(dst, "a.md")) != "original content" {
		t.Error("existing file should not be overwritten when force=false")
	}
}

// test opposite behaviors (overwrite vs skip) and necessarily share similar structure.
//
//nolint:dupl // TestSync_ForcesOverwriteWithForce and TestSync_SkipsExistingWithoutForce
func TestSync_ForcesOverwriteWithForce(t *testing.T) {
	src, dst := makeSrcDst(t)

	writeFile(t, filepath.Join(src, "a.md"), "new content")
	writeFile(t, filepath.Join(dst, "a.md"), "original content")

	res, err := dirsync.Sync(src, dst, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Forced) != 1 || res.Forced[0] != "a.md" {
		t.Errorf("Forced = %v; want [a.md]", res.Forced)
	}
	if len(res.Copied) != 0 {
		t.Errorf("Copied = %v; want empty", res.Copied)
	}
	if len(res.Skipped) != 0 {
		t.Errorf("Skipped = %v; want empty", res.Skipped)
	}
	// Verify new content was written.
	if readFile(t, filepath.Join(dst, "a.md")) != "new content" {
		t.Error("existing file should be overwritten when force=true")
	}
}

func TestSync_SourceNotExist(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "nonexistent")
	dst := filepath.Join(dir, "dst")

	res, err := dirsync.Sync(src, dst, false)
	if err != nil {
		t.Fatalf("expected nil error for missing src, got: %v", err)
	}

	if len(res.Copied) != 0 || len(res.Skipped) != 0 || len(res.Forced) != 0 {
		t.Errorf("expected empty result for missing src, got: %+v", res)
	}
}

func TestSync_CreatesDestDir(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "nested", "dst")

	if err := os.MkdirAll(src, 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(src, "file.md"), "content")

	res, err := dirsync.Sync(src, dst, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Copied) != 1 {
		t.Errorf("Copied = %v; want 1 entry", res.Copied)
	}

	if _, statErr := os.Stat(dst); statErr != nil {
		t.Errorf("expected dst to be created: %v", statErr)
	}
}

func TestSync_CopiesSubdirectory(t *testing.T) {
	src, dst := makeSrcDst(t)

	// Skill-style layout: <name>/SKILL.md
	if err := os.MkdirAll(filepath.Join(src, "jeff-skill-foo"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(src, "jeff-skill-foo", "SKILL.md"), "skill content")

	res, err := dirsync.Sync(src, dst, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Copied) != 1 || res.Copied[0] != "jeff-skill-foo" {
		t.Errorf("Copied = %v; want [jeff-skill-foo]", res.Copied)
	}

	got := readFile(t, filepath.Join(dst, "jeff-skill-foo", "SKILL.md"))
	if got != "skill content" {
		t.Errorf("SKILL.md content = %q; want %q", got, "skill content")
	}
}

// test opposite behaviors (skip vs overwrite on dirs) and necessarily share similar structure.
//
//nolint:dupl // TestSync_SkipsExistingSubdirectory and TestSync_ForcesExistingSubdirectory
func TestSync_SkipsExistingSubdirectory(t *testing.T) {
	src, dst := makeSrcDst(t)

	if err := os.MkdirAll(filepath.Join(src, "jeff-skill-foo"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(src, "jeff-skill-foo", "SKILL.md"), "new content")

	// Pre-existing directory in dst with different content.
	if err := os.MkdirAll(filepath.Join(dst, "jeff-skill-foo"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dst, "jeff-skill-foo", "SKILL.md"), "original content")

	res, err := dirsync.Sync(src, dst, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Skipped) != 1 || res.Skipped[0] != "jeff-skill-foo" {
		t.Errorf("Skipped = %v; want [jeff-skill-foo]", res.Skipped)
	}
	// Original content must be preserved.
	if readFile(t, filepath.Join(dst, "jeff-skill-foo", "SKILL.md")) != "original content" {
		t.Error("existing subdirectory should not be overwritten when force=false")
	}
}

// test opposite behaviors (overwrite vs skip on dirs) and necessarily share similar structure.
//
//nolint:dupl // TestSync_ForcesExistingSubdirectory and TestSync_SkipsExistingSubdirectory
func TestSync_ForcesExistingSubdirectory(t *testing.T) {
	src, dst := makeSrcDst(t)

	if err := os.MkdirAll(filepath.Join(src, "jeff-skill-foo"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(src, "jeff-skill-foo", "SKILL.md"), "new content")

	if err := os.MkdirAll(filepath.Join(dst, "jeff-skill-foo"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dst, "jeff-skill-foo", "SKILL.md"), "original content")

	res, err := dirsync.Sync(src, dst, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Forced) != 1 || res.Forced[0] != "jeff-skill-foo" {
		t.Errorf("Forced = %v; want [jeff-skill-foo]", res.Forced)
	}
	if readFile(t, filepath.Join(dst, "jeff-skill-foo", "SKILL.md")) != "new content" {
		t.Error("subdirectory content should be overwritten when force=true")
	}
}

func TestSync_ErrorOnUnreadableSourceFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test: running as root")
	}

	src, dst := makeSrcDst(t)

	srcFile := filepath.Join(src, "secret.md")
	writeFile(t, srcFile, "secret content")

	// Remove read permission so os.Open inside copyFile will fail.
	if err := os.Chmod(srcFile, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(srcFile, 0o600) })

	_, err := dirsync.Sync(src, dst, false)
	if err == nil {
		t.Fatal("expected error when source file is unreadable, got nil")
	}
}

func TestSync_ErrorOnUnreadableFileInSubdirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test: running as root")
	}

	src, dst := makeSrcDst(t)

	// Create a subdirectory with a file inside it.
	subDir := filepath.Join(src, "myskill")
	if err := os.MkdirAll(subDir, 0o750); err != nil {
		t.Fatal(err)
	}
	secretFile := filepath.Join(subDir, "secret.md")
	writeFile(t, secretFile, "secret content")

	// Remove all permissions from the file so copyFile will fail when
	// copyDir tries to open it.
	if err := os.Chmod(secretFile, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(secretFile, 0o600) })

	_, err := dirsync.Sync(src, dst, false)
	if err == nil {
		t.Fatal("expected error when file inside subdirectory is unreadable, got nil")
	}
}

func TestSync_ErrorWhenDestDirNotCreatable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test: running as root")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	// Put dst under a read-only directory so MkdirAll fails.
	parentDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(parentDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(src, 0o750); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(src, "file.md"), "content")

	if err := os.Chmod(parentDir, 0o500); err != nil { //nolint:gosec // 0o500 intentional to prevent subdirectory creation
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parentDir, 0o750) }) //nolint:gosec // restoring dir permissions for TempDir cleanup

	dst := filepath.Join(parentDir, "newdir")

	_, err := dirsync.Sync(src, dst, false)
	if err == nil {
		t.Fatal("expected error when dst cannot be created, got nil")
	}
}
