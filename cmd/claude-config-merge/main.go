// Package main is the entry point for the claude-config-merge CLI.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/jeff/claude-config-merge/internal/config"
)

func main() {
	// Global flags — must be parsed before the subcommand name.
	configPath := flag.String("config", config.DefaultPath(), "path to claude-config-merge config file")
	flag.Usage = func() { printUsage(os.Stderr) }
	flag.Parse()

	args := flag.Args()

	// No subcommand → show help.
	if len(args) == 0 {
		printUsage(os.Stdout)
		return
	}

	subcommand := args[0]
	args = args[1:]

	// Help subcommand needs no config.
	if subcommand == "help" {
		printUsage(os.Stdout)
		return
	}

	if *configPath == "" {
		log.Fatal("could not determine home directory; use -config to specify a config file path")
	}

	cfg, home := loadConfig(*configPath)

	if err := dispatch(subcommand, args, cfg, home, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// printUsage writes the full help text to w.
func printUsage(w io.Writer) {
	fmt.Fprintf(w, `claude-config-merge — sync Claude configuration from a master config directory

USAGE
  claude-config-merge [-config FILE] <command> [-f]

GLOBAL FLAGS
  -config FILE   Path to config file (default: ~/.claude-config-merge.json)
  -h             Show this help

CONFIG FILE
  Create ~/.claude-config-merge.json:
    {
      "configDir": "/path/to/your/claude/configs"
    }

  configDir mirrors the structure of ~. Expected layout inside configDir:
    <configDir>/.claude/settings.json   master settings (merged into ~/.claude/settings.json)
    <configDir>/.claude/agents/         agent files (synced to ~/.claude/agents/)
    <configDir>/.claude/skills/         skill files  (synced to ~/.claude/skills/)

COMMANDS
  settings    Merge master settings.json into ~/.claude/settings.json.
              New keys from master are added; existing local keys are kept.
              Use -f to let master values overwrite conflicting local keys.

  agents      Copy agent files from configDir/.claude/agents to ~/.claude/agents.
              Existing files are skipped unless -f is given.

  skills      Copy skill files from configDir/.claude/skills to ~/.claude/skills.
              Existing files are skipped unless -f is given.

  all         Run settings, agents, and skills in sequence.
              Accepts -f (applies to all three operations).

  cleanup-bak Delete settings.json.*.bak backup files from ~/.claude/.

  help        Show this help.

FLAGS (per command)
  -f          Force overwrite. For settings: master values win on conflict.
              For agents/skills/all: overwrite existing destination files.

EXAMPLES
  claude-config-merge settings
  claude-config-merge settings -f
  claude-config-merge agents
  claude-config-merge skills -f
  claude-config-merge all
  claude-config-merge all -f
  claude-config-merge cleanup-bak
  claude-config-merge -config ~/my-config.json all
`)
}

// dispatch executes the named subcommand using the provided config and home
// directory, writing output to w.
func dispatch(subcommand string, args []string, cfg *config.Config, home string, w io.Writer) error {
	switch subcommand {
	case "settings":
		force, err := parseForceFlagSet("settings", args)
		if err != nil {
			return err
		}
		masterPath := filepath.Join(cfg.ConfigDir, ".claude", "settings.json")
		localPath := filepath.Join(home, ".claude", "settings.json")
		return run(masterPath, localPath, force, w)

	case "agents":
		force, err := parseForceFlagSet("agents", args)
		if err != nil {
			return err
		}
		srcDir := filepath.Join(cfg.ConfigDir, ".claude", "agents")
		dstDir := filepath.Join(home, ".claude", "agents")
		return runSync(srcDir, dstDir, force, "Agents", w)

	case "skills":
		force, err := parseForceFlagSet("skills", args)
		if err != nil {
			return err
		}
		srcDir := filepath.Join(cfg.ConfigDir, ".claude", "skills")
		dstDir := filepath.Join(home, ".claude", "skills")
		return runSync(srcDir, dstDir, force, "Skills", w)

	case "all":
		force, err := parseForceFlagSet("all", args)
		if err != nil {
			return err
		}

		masterPath := filepath.Join(cfg.ConfigDir, ".claude", "settings.json")
		localPath := filepath.Join(home, ".claude", "settings.json")
		if err := run(masterPath, localPath, force, w); err != nil {
			return err
		}

		agentsSrc := filepath.Join(cfg.ConfigDir, ".claude", "agents")
		agentsDst := filepath.Join(home, ".claude", "agents")
		if err := runSync(agentsSrc, agentsDst, force, "Agents", w); err != nil {
			return err
		}

		skillsSrc := filepath.Join(cfg.ConfigDir, ".claude", "skills")
		skillsDst := filepath.Join(home, ".claude", "skills")
		return runSync(skillsSrc, skillsDst, force, "Skills", w)

	case "cleanup-bak":
		claudeDir := filepath.Join(home, ".claude")
		return runCleanupBak(claudeDir, w)

	default:
		fmt.Fprintf(w, "usage: claude-config-merge [settings|agents|skills|all|cleanup-bak] [-f]\n")
		return fmt.Errorf("unknown subcommand %q", subcommand)
	}
}

// loadConfig loads the tool config and resolves the home directory, exiting on
// any error.
func loadConfig(configPath string) (cfg *config.Config, home string) {
	var err error
	cfg, err = config.Load(configPath)
	if err != nil {
		log.Fatalf("error: %v\n\nCreate %s with contents:\n  {\"configDir\": \"/path/to/your/claude/configs\"}\n", err, configPath)
	}

	home, err = os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed to determine home directory: %v", err)
	}

	return cfg, home
}

// parseForceFlagSet parses a -f flag from args for the named subcommand and
// returns its value and any parse error.
func parseForceFlagSet(name string, args []string) (bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	f := fs.Bool("f", false, "overwrite existing files")
	if err := fs.Parse(args); err != nil {
		return false, fmt.Errorf("%s: %w", name, err)
	}
	return *f, nil
}
