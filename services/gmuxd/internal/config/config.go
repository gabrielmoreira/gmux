// Package config loads gmuxd configuration from ~/.config/gmux/config.toml.
//
// Missing file or missing keys are fine — everything has a safe default.
// The file is never written by gmuxd; users create and edit it manually.
package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is the top-level gmuxd configuration.
type Config struct {
	Port      int             `toml:"port"`
	Tailscale TailscaleConfig `toml:"tailscale"`
}

// TailscaleConfig controls the optional tailscale (tsnet) listener.
type TailscaleConfig struct {
	// Enabled starts a tsnet listener on the tailnet. Default false.
	Enabled bool `toml:"enabled"`

	// Hostname is the tailscale machine name (e.g. "gmux" → gmux.tailnet.ts.net).
	// Default "gmux".
	Hostname string `toml:"hostname"`

	// Allow is the list of tailscale login names permitted to connect
	// (e.g. "user@github"). Matched against the peer's UserProfile.LoginName.
	// Empty list = no one can connect (fail-closed).
	Allow []string `toml:"allow"`
}

// Load reads the config file. Returns defaults if the file doesn't exist.
func Load() Config {
	cfg := defaults()

	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("config: error reading %s: %v (using defaults)", path, err)
		}
		return cfg
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		log.Printf("config: error parsing %s: %v (using defaults)", path, err)
		return defaults()
	}

	// Normalize allow list entries.
	for i, entry := range cfg.Tailscale.Allow {
		cfg.Tailscale.Allow[i] = strings.TrimSpace(entry)
	}

	return cfg
}

func defaults() Config {
	return Config{
		Port: 8790,
		Tailscale: TailscaleConfig{
			Hostname: "gmux",
		},
	}
}

func configPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gmux", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gmux", "config.toml")
}
