package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gmuxapp/gmux/packages/paths"
)

func setTestStateEnv(t *testing.T) string {
	t.Helper()
	stateRoot := t.TempDir()
	t.Setenv("LOCALAPPDATA", stateRoot)
	t.Setenv("XDG_STATE_HOME", stateRoot)
	return stateRoot
}

// startTestSocketDaemon starts a minimal gmuxd on the standard test socket path.
func startTestSocketDaemon(t *testing.T, ver string) (cleanup func()) {
	t.Helper()
	setTestStateEnv(t)
	sockPath := paths.SocketPath()
	if err := os.MkdirAll(filepath.Dir(sockPath), 0o700); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"data": map[string]any{
				"service": "gmuxd",
				"version": ver,
				"status":  "ready",
			},
		})
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	time.Sleep(50 * time.Millisecond)

	return func() {
		srv.Close()
		os.Remove(sockPath)
	}
}

func TestGmuxdNeedsStart_NotRunning(t *testing.T) {
	old := version
	version = "0.4.4"
	defer func() { version = old }()

	setTestStateEnv(t)

	if !gmuxdNeedsStart() {
		t.Error("expected true when daemon is unreachable")
	}
}

func TestGmuxdNeedsStart_SameVersion(t *testing.T) {
	old := version
	version = "0.4.4"
	defer func() { version = old }()

	cleanup := startTestSocketDaemon(t, "0.4.4")
	defer cleanup()

	if gmuxdNeedsStart() {
		t.Error("expected false when versions match")
	}
}

func TestGmuxdNeedsStart_OlderVersion(t *testing.T) {
	old := version
	version = "0.4.4"
	defer func() { version = old }()

	cleanup := startTestSocketDaemon(t, "0.4.3")
	defer cleanup()

	if !gmuxdNeedsStart() {
		t.Error("expected true when daemon is older")
	}
}

func TestGmuxdNeedsStart_NewerVersion(t *testing.T) {
	old := version
	version = "0.4.3"
	defer func() { version = old }()

	cleanup := startTestSocketDaemon(t, "0.4.4")
	defer cleanup()

	if !gmuxdNeedsStart() {
		t.Error("expected true when versions differ")
	}
}

func TestGmuxdNeedsStart_DevNeverReplaces(t *testing.T) {
	old := version
	version = "dev"
	defer func() { version = old }()

	cleanup := startTestSocketDaemon(t, "0.4.3")
	defer cleanup()

	if gmuxdNeedsStart() {
		t.Error("dev builds must not replace a healthy daemon")
	}
}

func TestGmuxdNeedsStart_DevStartsWhenNotRunning(t *testing.T) {
	old := version
	version = "dev"
	defer func() { version = old }()

	setTestStateEnv(t)

	if !gmuxdNeedsStart() {
		t.Error("expected true for dev build when daemon is not running")
	}
}

func TestParseHealthField(t *testing.T) {
	body := []byte(`{"ok":true,"data":{"listen":"127.0.0.1:8790","auth_token":"abc123","version":"1.0.0"}}`)

	if got := parseHealthField(body, "listen"); got != "127.0.0.1:8790" {
		t.Errorf("listen = %q", got)
	}
	if got := parseHealthField(body, "auth_token"); got != "abc123" {
		t.Errorf("auth_token = %q", got)
	}
	if got := parseHealthField(body, "nonexistent"); got != "" {
		t.Errorf("nonexistent = %q, want empty", got)
	}
}

func TestResolveBundledBinaryFindsRepoBin(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "go.work"), []byte("go 1.26\n"), 0o644); err != nil {
		t.Fatalf("write go.work: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "cli", "gmux"), 0o755); err != nil {
		t.Fatalf("mkdir exe dir: %v", err)
	}
	want := filepath.Join(repoRoot, "bin", binaryFileName("gmuxd"))
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}
	if err := os.WriteFile(want, []byte("binary"), 0o644); err != nil {
		t.Fatalf("write gmuxd binary: %v", err)
	}

	self := filepath.Join(repoRoot, "cli", "gmux", binaryFileName("gmux"))
	if got := resolveBundledBinary(self, "gmuxd"); got != want {
		t.Fatalf("resolveBundledBinary() = %q, want %q", got, want)
	}
}

func TestResolveBundledBinaryPrefersSibling(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "go.work"), []byte("go 1.26\n"), 0o644); err != nil {
		t.Fatalf("write go.work: %v", err)
	}
	exeDir := filepath.Join(repoRoot, "custom-bin")
	if err := os.MkdirAll(exeDir, 0o755); err != nil {
		t.Fatalf("mkdir exe dir: %v", err)
	}
	repoBin := filepath.Join(repoRoot, "bin", binaryFileName("gmuxd"))
	if err := os.MkdirAll(filepath.Dir(repoBin), 0o755); err != nil {
		t.Fatalf("mkdir repo bin dir: %v", err)
	}
	sibling := filepath.Join(exeDir, binaryFileName("gmuxd"))
	if err := os.WriteFile(sibling, []byte("sibling"), 0o644); err != nil {
		t.Fatalf("write sibling binary: %v", err)
	}
	if err := os.WriteFile(repoBin, []byte("repo-bin"), 0o644); err != nil {
		t.Fatalf("write repo binary: %v", err)
	}

	self := filepath.Join(exeDir, binaryFileName("gmux"))
	if got := resolveBundledBinary(self, "gmuxd"); got != sibling {
		t.Fatalf("resolveBundledBinary() = %q, want sibling %q", got, sibling)
	}
}
