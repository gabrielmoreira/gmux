package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Point to a non-existent config dir.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg := Load()
	if cfg.Port != 8790 {
		t.Errorf("port = %d, want 8790", cfg.Port)
	}
	if cfg.Tailscale.Enabled {
		t.Error("tailscale should be disabled by default")
	}
	if cfg.Tailscale.Hostname != "gmux" {
		t.Errorf("hostname = %q, want %q", cfg.Tailscale.Hostname, "gmux")
	}
	if len(cfg.Tailscale.Allow) != 0 {
		t.Errorf("allow = %v, want empty", cfg.Tailscale.Allow)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "gmux")
	os.MkdirAll(cfgDir, 0o755)

	content := `
port = 9999

[tailscale]
enabled = true
hostname = "mybox"
allow = ["alice@github", "bob@github"]
`
	os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0o644)

	cfg := Load()
	if cfg.Port != 9999 {
		t.Errorf("port = %d, want 9999", cfg.Port)
	}
	if !cfg.Tailscale.Enabled {
		t.Error("tailscale should be enabled")
	}
	if cfg.Tailscale.Hostname != "mybox" {
		t.Errorf("hostname = %q, want %q", cfg.Tailscale.Hostname, "mybox")
	}
	if len(cfg.Tailscale.Allow) != 2 {
		t.Fatalf("allow = %v, want 2 entries", cfg.Tailscale.Allow)
	}
	if cfg.Tailscale.Allow[0] != "alice@github" {
		t.Errorf("allow[0] = %q", cfg.Tailscale.Allow[0])
	}
}

func TestLoadBadTOML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "gmux")
	os.MkdirAll(cfgDir, 0o755)
	os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte("{{invalid"), 0o644)

	cfg := Load()
	// Should return defaults, not crash.
	if cfg.Port != 8790 {
		t.Errorf("port = %d, want 8790 (defaults)", cfg.Port)
	}
}
