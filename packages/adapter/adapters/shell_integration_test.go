//go:build integration

// Shell adapter integration tests. Verifies the core WS→PTY input pipeline.
//
// Run: go test -tags integration -v -timeout 60s -run TestShell ./packages/adapter/adapters/

package adapters

import (
	"strings"
	"testing"
	"time"

	"github.com/gmuxapp/gmux/packages/adapter/adapters/testutil"
)

// TestShellWSInput verifies that WebSocket input reaches the PTY and produces
// output. This is the foundational test — if this fails, all adapter tests will too.
func TestShellWSInput(t *testing.T) {
	g := testutil.StartGmuxd(t)
	cwd := t.TempDir()

	sess := g.Launch([]string{"bash"}, cwd)
	send, _ := g.ConnectSession(sess.ID)
	g.WaitForOutput(sess.ID, 10*time.Second)

	send("echo GMUX_TEST_MARKER_42\r")

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		text := testutil.ReadScrollback(t, sess.SocketPath)
		if strings.Contains(text, "GMUX_TEST_MARKER_42") {
			t.Log("WS input verified — marker found in scrollback")
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	text := testutil.ReadScrollback(t, sess.SocketPath)
	t.Fatalf("marker not found in scrollback:\n%s", text)
}
