package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gmuxapp/gmux/packages/adapter/adapters"
	"github.com/gmuxapp/gmux/services/gmuxd/internal/store"
)

func TestSimilarityScoreExactMatch(t *testing.T) {
	score := similarityScore("hello world", "hello world")
	if score < 0.99 {
		t.Fatalf("expected ~1.0 for exact match, got %f", score)
	}
}

func TestSimilarityScorePartialMatch(t *testing.T) {
	// File tail is a substring of scrollback.
	score := similarityScore("fix the bug", "Let me fix the bug for you and also add tests")
	if score < 0.9 {
		t.Fatalf("expected high score for substring match, got %f", score)
	}
}

func TestSimilarityScoreNoMatch(t *testing.T) {
	score := similarityScore("aaaaa bbbbb ccccc", "xxxxx yyyyy zzzzz")
	if score > 0.2 {
		t.Fatalf("expected low score for no overlap, got %f", score)
	}
}

func TestSimilarityScoreEmpty(t *testing.T) {
	if similarityScore("", "hello") != 0 {
		t.Fatal("expected 0 for empty file tail")
	}
	if similarityScore("hello", "") != 0 {
		t.Fatal("expected 0 for empty scrollback")
	}
}

func TestLongestCommonSubstring(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"abcdef", "xbcdey", 4}, // "bcde"
		{"hello", "world", 1},   // "l" or "o"
		{"", "abc", 0},
		{"same", "same", 4},
		{"abc", "xyz", 0},
	}
	for _, tt := range tests {
		got := longestCommonSubstring(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("lcs(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestTail(t *testing.T) {
	if tail("hello world", 5) != "world" {
		t.Fatal("expected 'world'")
	}
	if tail("hi", 10) != "hi" {
		t.Fatal("expected 'hi' when n > len")
	}
}

func TestNotifyNewSessionDoesNotStealTitleFromOldPiFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cwd := "/home/user/dev/project"
	pi := adapters.NewPi()
	sessionDir := pi.SessionDir(cwd)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}

	oldFile := filepath.Join(sessionDir, "2026-03-16T10-00-00-000Z_old.jsonl")
	oldContent := "" +
		"{\"type\":\"session\",\"id\":\"old-session\",\"cwd\":\"/home/user/dev/project\",\"timestamp\":\"2026-03-16T10:00:00Z\"}\n" +
		"{\"type\":\"message\",\"role\":\"user\",\"text\":\"fix auth bug\"}\n"
	if err := os.WriteFile(oldFile, []byte(oldContent), 0o644); err != nil {
		t.Fatalf("write old jsonl: %v", err)
	}

	s := store.New()
	s.Upsert(store.Session{
		ID:         "sess-new",
		Cwd:        cwd,
		Kind:       "pi",
		Alive:      true,
		Title:      "pi",
		SocketPath: "/tmp/gmux-sessions/sess-new.sock",
	})

	fm := NewFileMonitor(s)
	if fm.watcher != nil {
		defer fm.watcher.Close()
	}

	fm.NotifyNewSession("sess-new")

	// Before the fix, NotifyNewSession eagerly attributed the most recent JSONL
	// file in the directory, then asynchronously parsed it and overwrote the new
	// session title with the old session's first user message.
	time.Sleep(700 * time.Millisecond)

	sess, ok := s.Get("sess-new")
	if !ok {
		t.Fatal("session disappeared")
	}
	if sess.Title != "pi" {
		t.Fatalf("title = %q, want %q (do not steal old session title)", sess.Title, "pi")
	}
	if len(fm.attributions) != 0 {
		t.Fatalf("attributions = %v, want none until a real file write occurs", fm.attributions)
	}
}
