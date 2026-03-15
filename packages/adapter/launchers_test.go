package adapter_test

import (
	"testing"

	"github.com/gmuxapp/gmux/packages/adapter/adapters"
)

func TestAllLaunchersIncludesPiAndShell(t *testing.T) {
	launchers := adapters.AllLaunchers()
	if len(launchers) < 2 {
		t.Fatalf("expected at least 2 launchers, got %d", len(launchers))
	}

	ids := make([]string, 0, len(launchers))
	seen := map[string]bool{}
	for _, l := range launchers {
		ids = append(ids, l.ID)
		if seen[l.ID] {
			t.Fatalf("duplicate launcher id %q in %v", l.ID, ids)
		}
		seen[l.ID] = true
	}

	if !seen["pi"] {
		t.Fatalf("expected pi launcher in %v", ids)
	}
	if !seen["shell"] {
		t.Fatalf("expected shell launcher in %v", ids)
	}
	if got := launchers[len(launchers)-1].ID; got != "shell" {
		t.Fatalf("expected shell last, got %q in %v", got, ids)
	}
}
