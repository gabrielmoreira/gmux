package adapters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gmuxapp/gmux/packages/adapter"
)

// --- Shell adapter tests ---

func TestShellMatchAll(t *testing.T) {
	g := NewShell()
	if !g.Match([]string{"anything"}) {
		t.Fatal("shell should match any command")
	}
	if !g.Match([]string{}) {
		t.Fatal("shell should match empty command")
	}
}

func TestShellName(t *testing.T) {
	g := NewShell()
	if g.Name() != "shell" {
		t.Fatalf("expected 'shell', got %q", g.Name())
	}
}

func TestShellEnvNil(t *testing.T) {
	g := NewShell()
	env := g.Env(adapter.EnvContext{})
	if env != nil {
		t.Fatalf("expected nil env, got %v", env)
	}
}

func TestShellMonitorPlainOutput(t *testing.T) {
	g := NewShell()
	if g.Monitor([]byte("hello")) != nil {
		t.Fatal("shell should not report status for plain output")
	}
}

// --- OSC title parsing tests ---

func TestParseOSCTitleBEL(t *testing.T) {
	data := []byte("\x1b]0;my title\x07 more data")
	if title := parseOSCTitle(data); title != "my title" {
		t.Fatalf("expected 'my title', got %q", title)
	}
}

func TestParseOSCTitleST(t *testing.T) {
	data := []byte("\x1b]2;window title\x1b\\ more")
	if title := parseOSCTitle(data); title != "window title" {
		t.Fatalf("expected 'window title', got %q", title)
	}
}

func TestParseOSCTitleNone(t *testing.T) {
	if title := parseOSCTitle([]byte("hello world")); title != "" {
		t.Fatalf("expected empty, got %q", title)
	}
}

func TestParseOSCTitleEmbedded(t *testing.T) {
	data := []byte("some output\r\n\x1b]0;~/dev/gmux\x07prompt $ ")
	if title := parseOSCTitle(data); title != "~/dev/gmux" {
		t.Fatalf("expected '~/dev/gmux', got %q", title)
	}
}

func TestShellMonitorTitleUpdate(t *testing.T) {
	g := NewShell()
	status := g.Monitor([]byte("\x1b]0;fish: ~/dev\x07"))
	if status == nil {
		t.Fatal("should return status")
	}
	if status.Title != "fish: ~/dev" {
		t.Fatalf("expected title 'fish: ~/dev', got %q", status.Title)
	}
	if status.State != "" {
		t.Fatalf("expected no state, got %q", status.State)
	}
}

// --- Pi adapter tests ---

func TestPiName(t *testing.T) {
	if NewPi().Name() != "pi" {
		t.Fatal("expected 'pi'")
	}
}

func TestPiMatchDirect(t *testing.T) {
	p := NewPi()
	if !p.Match([]string{"pi"}) {
		t.Fatal("should match 'pi'")
	}
	if !p.Match([]string{"pi-coding-agent"}) {
		t.Fatal("should match 'pi-coding-agent'")
	}
}

func TestPiMatchWrapped(t *testing.T) {
	p := NewPi()
	if !p.Match([]string{"npx", "pi"}) {
		t.Fatal("should match 'npx pi'")
	}
	if !p.Match([]string{"env", "pi", "--flag"}) {
		t.Fatal("should match 'env pi --flag'")
	}
	if !p.Match([]string{"/home/user/.local/bin/pi"}) {
		t.Fatal("should match full path")
	}
}

func TestPiMatchStopsAtDoubleDash(t *testing.T) {
	p := NewPi()
	if p.Match([]string{"echo", "--", "pi"}) {
		t.Fatal("should not match 'pi' after '--'")
	}
}

func TestPiNoMatchOther(t *testing.T) {
	p := NewPi()
	if p.Match([]string{"pytest", "tests/"}) {
		t.Fatal("should not match pytest")
	}
	if p.Match([]string{"pipeline"}) {
		t.Fatal("should not match 'pipeline'")
	}
}

func TestPiEnvNil(t *testing.T) {
	if env := NewPi().Env(adapter.EnvContext{}); env != nil {
		t.Fatalf("expected nil, got %v", env)
	}
}

func TestPiMonitorPlainOutput(t *testing.T) {
	if NewPi().Monitor([]byte("some output")) != nil {
		t.Fatal("should return nil for non-spinner output")
	}
}

func TestPiMonitorSpinner(t *testing.T) {
	s := NewPi().Monitor([]byte("⠋ Working..."))
	if s == nil {
		t.Fatal("should detect spinner")
	}
	if s.State != "active" || s.Label != "working" {
		t.Fatalf("expected active/working, got %s/%s", s.State, s.Label)
	}
}

// --- Pi capability interface checks ---

func TestPiImplementsSessionFiler(t *testing.T) {
	var a adapter.Adapter = NewPi()
	if _, ok := a.(adapter.SessionFiler); !ok {
		t.Fatal("Pi should implement SessionFiler")
	}
}

func TestPiImplementsFileMonitor(t *testing.T) {
	var a adapter.Adapter = NewPi()
	if _, ok := a.(adapter.FileMonitor); !ok {
		t.Fatal("Pi should implement FileMonitor")
	}
}

func TestPiImplementsResumer(t *testing.T) {
	var a adapter.Adapter = NewPi()
	if _, ok := a.(adapter.Resumer); !ok {
		t.Fatal("Pi should implement Resumer")
	}
}

func TestShellDoesNotImplementCapabilities(t *testing.T) {
	var a adapter.Adapter = NewShell()
	if _, ok := a.(adapter.SessionFiler); ok {
		t.Fatal("Shell should not implement SessionFiler")
	}
	if _, ok := a.(adapter.Resumer); ok {
		t.Fatal("Shell should not implement Resumer")
	}
}

// --- Pi session file tests ---

func writeTempJSONL(t *testing.T, lines ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-session.jsonl")
	var content string
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseSessionFileFirstUserMessage(t *testing.T) {
	path := writeTempJSONL(t,
		`{"type":"session","version":3,"id":"abc-123","timestamp":"2026-03-15T10:00:00Z","cwd":"/tmp/test"}`,
		`{"type":"model_change","id":"m1","timestamp":"2026-03-15T10:00:00Z"}`,
		`{"type":"message","id":"u1","timestamp":"2026-03-15T10:01:00Z","message":{"role":"user","content":[{"type":"text","text":"Fix the auth bug in login.go"}]}}`,
		`{"type":"message","id":"a1","timestamp":"2026-03-15T10:01:05Z","message":{"role":"assistant","content":[{"type":"text","text":"I'll fix that for you."}]}}`,
	)

	p := NewPi()
	info, err := p.ParseSessionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "abc-123" {
		t.Errorf("expected id abc-123, got %s", info.ID)
	}
	if info.Cwd != "/tmp/test" {
		t.Errorf("expected cwd /tmp/test, got %s", info.Cwd)
	}
	if info.Title != "Fix the auth bug in login.go" {
		t.Errorf("expected first user msg as title, got %q", info.Title)
	}
	if info.MessageCount != 2 {
		t.Errorf("expected 2 messages, got %d", info.MessageCount)
	}
}

func TestParseSessionFileNameOverridesFirstMessage(t *testing.T) {
	path := writeTempJSONL(t,
		`{"type":"session","version":3,"id":"abc-456","timestamp":"2026-03-15T10:00:00Z","cwd":"/tmp/test"}`,
		`{"type":"message","id":"u1","timestamp":"2026-03-15T10:01:00Z","message":{"role":"user","content":[{"type":"text","text":"Fix the auth bug"}]}}`,
		`{"type":"session_info","name":"  Auth refactor  "}`,
		`{"type":"message","id":"a1","timestamp":"2026-03-15T10:01:05Z","message":{"role":"assistant","content":[{"type":"text","text":"Done."}]}}`,
	)

	info, err := NewPi().ParseSessionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Auth refactor" {
		t.Errorf("expected session_info name as title, got %q", info.Title)
	}
}

func TestParseSessionFileNoMessages(t *testing.T) {
	path := writeTempJSONL(t,
		`{"type":"session","version":3,"id":"abc-789","timestamp":"2026-03-15T10:00:00Z","cwd":"/tmp/test"}`,
	)

	info, err := NewPi().ParseSessionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "(new)" {
		t.Errorf("expected fallback title, got %q", info.Title)
	}
}

func TestParseSessionFileLongTitleTruncated(t *testing.T) {
	long := "Please help me with this very long request that goes on and on about many different things and really should be truncated for the sidebar"
	path := writeTempJSONL(t,
		`{"type":"session","version":3,"id":"abc-long","timestamp":"2026-03-15T10:00:00Z","cwd":"/tmp/test"}`,
		`{"type":"message","id":"u1","timestamp":"2026-03-15T10:01:00Z","message":{"role":"user","content":[{"type":"text","text":"`+long+`"}]}}`,
	)

	info, err := NewPi().ParseSessionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Title) > 85 {
		t.Errorf("title too long (%d chars): %q", len(info.Title), info.Title)
	}
	if info.Title[len(info.Title)-3:] != "…" {
		t.Errorf("expected truncation marker, got %q", info.Title)
	}
}

func TestParseSessionFileStringContent(t *testing.T) {
	path := writeTempJSONL(t,
		`{"type":"session","version":3,"id":"abc-str","timestamp":"2026-03-15T10:00:00Z","cwd":"/tmp/test"}`,
		`{"type":"message","id":"u1","timestamp":"2026-03-15T10:01:00Z","message":{"role":"user","content":"Help me debug this"}}`,
	)

	info, err := NewPi().ParseSessionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Help me debug this" {
		t.Errorf("expected string content as title, got %q", info.Title)
	}
}

// --- Pi FileMonitor tests ---

func TestParseNewLinesNameChange(t *testing.T) {
	p := NewPi()
	events := p.ParseNewLines([]string{
		`{"type":"session_info","name":"My new name"}`,
		`{"type":"message","id":"u1","message":{"role":"user","content":"hello"}}`,
	})
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "My new name" {
		t.Errorf("expected title 'My new name', got %q", events[0].Title)
	}
}

func TestParseNewLinesNoEvents(t *testing.T) {
	p := NewPi()
	events := p.ParseNewLines([]string{
		`{"type":"message","id":"u1","message":{"role":"user","content":"hello"}}`,
	})
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

// --- Pi Resumer tests ---

func TestResumeCommand(t *testing.T) {
	p := NewPi()
	cmd := p.ResumeCommand(&adapter.SessionFileInfo{
		FilePath: "/home/user/.pi/agent/sessions/--tmp--/test.jsonl",
	})
	expected := []string{"pi", "--session", "/home/user/.pi/agent/sessions/--tmp--/test.jsonl", "-c"}
	if len(cmd) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, cmd)
	}
	for i := range cmd {
		if cmd[i] != expected[i] {
			t.Errorf("cmd[%d]: expected %q, got %q", i, expected[i], cmd[i])
		}
	}
}

func TestCanResumeValid(t *testing.T) {
	path := writeTempJSONL(t,
		`{"type":"session","version":3,"id":"abc","timestamp":"2026-03-15T10:00:00Z","cwd":"/tmp/test"}`,
		`{"type":"message","id":"u1","timestamp":"2026-03-15T10:01:00Z","message":{"role":"user","content":[{"type":"text","text":"hello"}]}}`,
	)
	if !NewPi().CanResume(path) {
		t.Fatal("should be resumable")
	}
}

func TestCanResumeEmpty(t *testing.T) {
	path := writeTempJSONL(t,
		`{"type":"session","version":3,"id":"abc","timestamp":"2026-03-15T10:00:00Z","cwd":"/tmp/test"}`,
	)
	if NewPi().CanResume(path) {
		t.Fatal("empty session should not be resumable")
	}
}

// --- Shared helpers ---

func TestSessionDirEncoding(t *testing.T) {
	p := NewPi()
	dir := p.SessionDir("/home/mg/dev/gmux")
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %s", dir)
	}
	if base := filepath.Base(dir); base != "--home-mg-dev-gmux--" {
		t.Errorf("expected --home-mg-dev-gmux--, got %s", base)
	}
}

func TestListSessionFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "b.jsonl"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("not a session"), 0644)

	if files := ListSessionFiles(dir); len(files) != 2 {
		t.Errorf("expected 2 jsonl files, got %d", len(files))
	}
}
