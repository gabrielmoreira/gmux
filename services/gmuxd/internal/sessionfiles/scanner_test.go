package sessionfiles

import (
	"testing"
	"time"

	"github.com/gmuxapp/gmux/services/gmuxd/internal/store"
)

func TestPurgeStaleSessions(t *testing.T) {
	s := store.New()

	s.Upsert(store.Session{
		ID:       "stale",
		Alive:    false,
		ExitedAt: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
	})
	s.Upsert(store.Session{
		ID:       "fresh",
		Alive:    false,
		ExitedAt: time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339),
	})
	s.Upsert(store.Session{
		ID:       "resumable",
		Alive:    false,
		Slug:     "some-key",
		ExitedAt: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
	})

	sc := New(s)
	sc.PurgeStaleSessions(1 * time.Hour)

	ids := map[string]bool{}
	for _, sess := range s.List() {
		ids[sess.ID] = true
	}

	if ids["stale"] {
		t.Error("stale session should have been purged")
	}
	if !ids["fresh"] {
		t.Error("fresh session should still be present")
	}
	if !ids["resumable"] {
		t.Error("resumable session should still be present")
	}
}

func TestScanHandlesShortSessionIDs(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(tmpHome, ".pi", "agent"))

	writePiSession(t, tmpHome, "/tmp/project-short", "abc", "hello")

	s := newTestStore()
	sc := New(s)
	sc.Scan()

	var found *store.Session
	for _, sess := range s.List() {
		if sess.ResumeKey == "abc" {
			copy := sess
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatal("expected session with resume key abc to be discovered")
	}
	if found.ID != "file-abc" {
		t.Fatalf("expected short session id to remain intact, got %q", found.ID)
	}
	if found.ResumeKey != "abc" {
		t.Fatalf("expected resume key abc, got %q", found.ResumeKey)
	}
}

func TestScanSkipsEmptySessionIDs(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(tmpHome, ".pi", "agent"))

	writePiSession(t, tmpHome, "/tmp/project-empty", "", "hello")

	s := newTestStore()
	sc := New(s)
	sc.Scan()

	for _, sess := range s.List() {
		if sess.ID == "file-" || sess.ResumeKey == "" {
			t.Fatalf("expected empty session ids to be skipped, found %#v", sess)
		}
	}
}
