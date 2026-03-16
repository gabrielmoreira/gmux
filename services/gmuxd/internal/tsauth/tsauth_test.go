package tsauth

import "testing"

func TestIsAllowed(t *testing.T) {
	l := &Listener{
		cfg: Config{
			Allow: []string{"alice@github", "bobs-phone"},
		},
	}

	tests := []struct {
		login, node string
		want        bool
	}{
		{"alice@github", "alices-laptop", true},   // login match
		{"bob@github", "bobs-phone", true},         // device match
		{"eve@github", "eves-laptop", false},       // no match
		{"Alice@GitHub", "whatever", true},          // case-insensitive login
		{"whoever", "Bobs-Phone", true},             // case-insensitive device
		{"", "", false},                              // empty
	}

	for _, tt := range tests {
		got := l.isAllowed(tt.login, tt.node)
		if got != tt.want {
			t.Errorf("isAllowed(%q, %q) = %v, want %v", tt.login, tt.node, got, tt.want)
		}
	}
}

func TestIsAllowedEmptyList(t *testing.T) {
	l := &Listener{
		cfg: Config{Allow: nil},
	}

	if l.isAllowed("anyone@github", "any-device") {
		t.Error("empty allow list should deny everyone")
	}
}
