// Package main is the entry point for the claude-config-merge CLI.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jeff/claude-config-merge/internal/config"
)

func main() {
	configPath := flag.String("config", config.DefaultPath(), "path to claude-config-merge config file")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("could not determine home directory; use -config to specify a config file path")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("error: %v\n\nCreate %s with contents:\n  {\"configDir\": \"/path/to/your/claude/configs\"}\n", err, *configPath)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed to determine home directory: %v", err)
	}

	masterPath := filepath.Join(cfg.ConfigDir, ".claude", "settings.json")
	localPath := filepath.Join(home, ".claude", "settings.json")

	if err := run(masterPath, localPath, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
