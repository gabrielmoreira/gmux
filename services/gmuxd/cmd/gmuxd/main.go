package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type session struct {
	SessionID  string  `json:"session_id"`
	AbducoName string  `json:"abduco_name"`
	Title      string  `json:"title,omitempty"`
	Kind       string  `json:"kind"`
	State      string  `json:"state"`
	UpdatedAt  float64 `json:"updated_at"`
}

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]session
}

func newSessionStore() *sessionStore {
	now := nowUnix()
	return &sessionStore{
		sessions: map[string]session{
			"sess-1": {
				SessionID:  "sess-1",
				AbducoName: "pi:gmux:1",
				Title:      "gmux bootstrap",
				Kind:       "pi",
				State:      "running",
				UpdatedAt:  now,
			},
			"sess-2": {
				SessionID:  "sess-2",
				AbducoName: "pi:gmux:2",
				Title:      "docs cleanup",
				Kind:       "pi",
				State:      "waiting",
				UpdatedAt:  now - 15,
			},
		},
	}
}

func (s *sessionStore) list() []session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]session, 0, len(s.sessions))
	for _, item := range s.sessions {
		items = append(items, item)
	}
	return items
}

func (s *sessionStore) toggleState(id string) (session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.sessions[id]
	if !ok {
		return session{}, false
	}

	if item.State == "running" {
		item.State = "waiting"
	} else {
		item.State = "running"
	}
	item.UpdatedAt = nowUnix()
	s.sessions[id] = item
	return item, true
}

func main() {
	store := newSessionStore()
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ok": true,
			"data": map[string]any{
				"service": "gmuxd",
				"node_id": "node-local",
			},
		})
	})

	mux.HandleFunc("/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		writeJSON(w, map[string]any{
			"ok":   true,
			"data": store.list(),
		})
	})

	mux.HandleFunc("/v1/sessions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !strings.HasSuffix(r.URL.Path, "/attach") {
			http.NotFound(w, r)
			return
		}

		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 4 {
			http.NotFound(w, r)
			return
		}

		sessionID := parts[2]
		writeJSON(w, map[string]any{
			"ok": true,
			"data": map[string]any{
				"transport": "ttyd",
				"port":      7711,
				"is_new":    sessionID == "sess-1",
			},
		})
	})

	mux.HandleFunc("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		sendEvent(w, "session-upsert", map[string]any{
			"type":       "session-upsert",
			"session_id": "sess-1",
			"updated_at": nowUnix(),
			"session": map[string]any{
				"session_id":  "sess-1",
				"abduco_name": "pi:gmux:1",
				"title":       "gmux bootstrap",
				"kind":        "pi",
				"state":       "running",
				"updated_at":  nowUnix(),
			},
		})
		flusher.Flush()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		notify := r.Context().Done()
		for {
			select {
			case <-notify:
				return
			case <-ticker.C:
				updated, found := store.toggleState("sess-1")
				if !found {
					continue
				}
				sendEvent(w, "session-state", map[string]any{
					"type":       "session-state",
					"session_id": updated.SessionID,
					"state":      updated.State,
					"updated_at": updated.UpdatedAt,
				})
				fmt.Fprint(w, ": keepalive\n\n")
				flusher.Flush()
			}
		}
	})

	addr := envOr("GMUXD_ADDR", ":8790")
	log.Printf("gmuxd listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func sendEvent(w http.ResponseWriter, event string, payload any) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", bytes)
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func nowUnix() float64 {
	return float64(time.Now().UnixNano()) / float64(time.Second)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
