package adapters

import (
	"bytes"
	"path/filepath"

	"github.com/gmuxapp/gmux/cli/gmuxr/internal/adapter"
)

func init() {
	Launchers = append(Launchers, Launcher{
		ID:          "pi",
		Label:       "pi",
		Command:     []string{"pi"},
		Description: "Coding agent",
	})
	All = append(All, NewPi())
}

// Pi is the adapter for the pi coding agent.
// Recognizes pi/pi-coding-agent commands and monitors PTY output for
// spinner patterns to report active/idle status.
//
// Session file attribution (resume key) is handled by gmuxd, not here.
// See ADR-0009 for the content-similarity matching design.
type Pi struct{}

func NewPi() *Pi { return &Pi{} }

func (p *Pi) Name() string { return "pi" }

// Match returns true if any argument in the command is the `pi` or
// `pi-coding-agent` binary.
func (p *Pi) Match(cmd []string) bool {
	for _, arg := range cmd {
		base := filepath.Base(arg)
		if base == "pi" || base == "pi-coding-agent" {
			return true
		}
		if arg == "--" {
			break
		}
	}
	return false
}

// Env returns no extra environment variables.
func (p *Pi) Env(ctx adapter.EnvContext) []string {
	return nil
}

// Spinner characters used by pi's TUI (braille pattern dots).
var piSpinnerChars = [][]byte{
	[]byte("⠋"), []byte("⠙"), []byte("⠹"), []byte("⠸"),
	[]byte("⠼"), []byte("⠴"), []byte("⠦"), []byte("⠧"),
	[]byte("⠇"), []byte("⠏"),
}

// Monitor detects pi's spinner pattern in PTY output.
// When a spinner character followed by " Working..." is detected,
// reports active status. Returns nil for non-spinner output.
func (p *Pi) Monitor(output []byte) *adapter.Status {
	for _, sc := range piSpinnerChars {
		if idx := bytes.Index(output, sc); idx >= 0 {
			// Check if "Working..." follows the spinner
			rest := output[idx+len(sc):]
			if bytes.Contains(rest, []byte("Working")) {
				return &adapter.Status{
					Label: "working",
					State: "active",
				}
			}
		}
	}
	return nil
}
