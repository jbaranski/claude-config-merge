package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeff/claude-config-merge/internal/config"
)

// makeConfig creates a test Config and directory structure.
// It returns the cfg, the configDir (acting as a remote), and a homeDir.
func makeConfig(t *testing.T) (cfg *config.Config, configDir, homeDir string) {
	t.Helper()
	base := t.TempDir()
	configDir = filepath.Join(base, "remote")
	homeDir = filepath.Join(base, "home")

	for _, d := range []string{
		filepath.Join(configDir, ".claude"),
		filepath.Join(homeDir, ".claude"),
	} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	cfg = &config.Config{ConfigDir: configDir}
	return cfg, configDir, homeDir
}

// ---- dispatch tests ----

func TestDispatch_Settings(t *testing.T) {
	cfg, configDir, homeDir := makeConfig(t)

	masterPath := filepath.Join(configDir, ".claude", "settings.json")
	localPath := filepath.Join(homeDir, ".claude", "settings.json")
	writeJSON(t, masterPath, map[string]any{"key": "value"})
	writeJSON(t, localPath, map[string]any{})

	var buf bytes.Buffer
	if err := dispatch("settings", nil, cfg, homeDir, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatch_SettingsDefault(t *testing.T) {
	cfg, configDir, homeDir := makeConfig(t)

	masterPath := filepath.Join(configDir, ".claude", "settings.json")
	localPath := filepath.Join(homeDir, ".claude", "settings.json")
	writeJSON(t, masterPath, map[string]any{"fromMaster": "yes"})
	writeJSON(t, localPath, map[string]any{})

	var buf bytes.Buffer
	// Empty subcommand args — default is "settings".
	if err := dispatch("settings", []string{}, cfg, homeDir, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "fromMaster") {
		t.Errorf("expected merged key in output, got:\n%s", output)
	}
}

// that necessarily have similar structure (create dir, dispatch, check output).
//
//nolint:dupl // TestDispatch_Agents and TestDispatch_Skills test separate subcommands
func TestDispatch_Agents(t *testing.T) {
	cfg, configDir, homeDir := makeConfig(t)

	// Create an agent file in the "remote" configDir.
	agentsSrc := filepath.Join(configDir, ".claude", "agents")
	if err := os.MkdirAll(agentsSrc, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentsSrc, "my-agent.md"), []byte("agent"), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := dispatch("agents", []string{}, cfg, homeDir, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "my-agent.md") {
		t.Errorf("expected agent file in output, got:\n%s", output)
	}
}

// that necessarily have similar structure (create dir, dispatch, check output).
//
//nolint:dupl // TestDispatch_Skills and TestDispatch_Agents test separate subcommands
func TestDispatch_Skills(t *testing.T) {
	cfg, configDir, homeDir := makeConfig(t)

	skillsSrc := filepath.Join(configDir, ".claude", "skills")
	if err := os.MkdirAll(skillsSrc, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsSrc, "my-skill.md"), []byte("skill"), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := dispatch("skills", []string{}, cfg, homeDir, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "my-skill.md") {
		t.Errorf("expected skill file in output, got:\n%s", output)
	}
}

func TestDispatch_All(t *testing.T) {
	cfg, configDir, homeDir := makeConfig(t)

	masterPath := filepath.Join(configDir, ".claude", "settings.json")
	localPath := filepath.Join(homeDir, ".claude", "settings.json")
	writeJSON(t, masterPath, map[string]any{"newKey": "value"})
	writeJSON(t, localPath, map[string]any{})

	agentsSrc := filepath.Join(configDir, ".claude", "agents")
	if err := os.MkdirAll(agentsSrc, 0o750); err != nil {
		t.Fatal(err)
	}

	skillsSrc := filepath.Join(configDir, ".claude", "skills")
	if err := os.MkdirAll(skillsSrc, 0o750); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := dispatch("all", []string{}, cfg, homeDir, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatch_AllWithForce(t *testing.T) {
	cfg, configDir, homeDir := makeConfig(t)

	masterPath := filepath.Join(configDir, ".claude", "settings.json")
	localPath := filepath.Join(homeDir, ".claude", "settings.json")
	writeJSON(t, masterPath, map[string]any{"k": "v"})
	writeJSON(t, localPath, map[string]any{})

	var buf bytes.Buffer
	if err := dispatch("all", []string{"-f"}, cfg, homeDir, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDispatch_UnknownSubcommand(t *testing.T) {
	cfg, _, homeDir := makeConfig(t)

	var buf bytes.Buffer
	err := dispatch("bogus", nil, cfg, homeDir, &buf)
	if err == nil {
		t.Fatal("expected error for unknown subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Errorf("error = %q; want to contain 'unknown subcommand'", err.Error())
	}
	// Output (stdout) must contain the usage hint but NOT repeat the error message.
	output := buf.String()
	if !strings.Contains(output, "usage:") {
		t.Errorf("expected usage message in output, got:\n%s", output)
	}
	if strings.Contains(output, "unknown subcommand") {
		t.Errorf("error message must not be printed to stdout; got:\n%s", output)
	}
}

// ---- parseForceFlagSet tests ----

func TestParseForceFlagSet_DefaultFalse(t *testing.T) {
	got, err := parseForceFlagSet("test", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false when -f not provided, got true")
	}
}

func TestParseForceFlagSet_TrueWhenFlagSet(t *testing.T) {
	got, err := parseForceFlagSet("test", []string{"-f"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected true when -f is provided, got false")
	}
}

func TestParseForceFlagSet_UnknownFlag(t *testing.T) {
	_, err := parseForceFlagSet("test", []string{"-unknown"})
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
	if !strings.Contains(err.Error(), "test") {
		t.Errorf("error = %q; want to contain subcommand name 'test'", err.Error())
	}
}

// ---- dirExists tests ----

func TestDirExists_ExistingDir(t *testing.T) {
	dir := t.TempDir()
	if !dirExists(dir) {
		t.Errorf("dirExists(%q) = false; want true for existing directory", dir)
	}
}

func TestDirExists_NonExistent(t *testing.T) {
	dir := t.TempDir()
	missing := dir + "/nonexistent"
	if dirExists(missing) {
		t.Errorf("dirExists(%q) = true; want false for nonexistent path", missing)
	}
}

func TestDirExists_File(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/file.txt"
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if dirExists(path) {
		t.Errorf("dirExists(%q) = true; want false for a file path", path)
	}
}
