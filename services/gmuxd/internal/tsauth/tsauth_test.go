package tsauth

import "testing"

func TestIsAllowed(t *testing.T) {
	l := &Listener{
		cfg: Config{
			Allow: []string{"alice@github", "bob@github"},
		},
	}

	tests := []struct {
		login string
		want  bool
	}{
		{"alice@github", true},    // exact match
		{"bob@github", true},      // exact match
		{"eve@github", false},     // no match
		{"Alice@GitHub", true},    // case-insensitive
		{"", false},               // empty
	}

	for _, tt := range tests {
		got := l.isAllowed(tt.login)
		if got != tt.want {
			t.Errorf("isAllowed(%q) = %v, want %v", tt.login, got, tt.want)
		}
	}
}

func TestIsAllowedEmptyList(t *testing.T) {
	l := &Listener{
		cfg: Config{Allow: nil},
	}

	if l.isAllowed("anyone@github") {
		t.Error("empty allow list should deny everyone")
	}
}
