package main

import "testing"

func TestDiscoverLaunchersUsesCompiledAdapters(t *testing.T) {
	cfg := discoverLaunchers()
	if cfg.DefaultLauncher != "shell" {
		t.Fatalf("expected default launcher shell, got %q", cfg.DefaultLauncher)
	}
	if len(cfg.Launchers) < 2 {
		t.Fatalf("expected at least 2 launchers, got %d", len(cfg.Launchers))
	}

	seenPi := false
	for _, l := range cfg.Launchers {
		if l.ID == "pi" {
			seenPi = true
			break
		}
	}
	if !seenPi {
		t.Fatalf("expected pi launcher in %#v", cfg.Launchers)
	}
	if got := cfg.Launchers[len(cfg.Launchers)-1].ID; got != "shell" {
		t.Fatalf("expected shell last, got %q", got)
	}
}
