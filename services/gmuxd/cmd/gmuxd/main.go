package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gmuxapp/gmux/services/gmuxd/internal/discovery"
	"github.com/gmuxapp/gmux/services/gmuxd/internal/store"
	"github.com/gmuxapp/gmux/services/gmuxd/internal/wsproxy"
)

type Launcher struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Command     []string `json:"command"`
	Description string   `json:"description,omitempty"`
}

type LaunchConfig struct {
	DefaultLauncher string     `json:"default_launcher"`
	Launchers       []Launcher `json:"launchers"`
}

// discoverLaunchers calls "gmuxr adapters" to get the adapter-registered
// launchers, then prepends shell as the default.
func discoverLaunchers(gmuxrPath string) LaunchConfig {
	cfg := LaunchConfig{
		DefaultLauncher: "shell",
		Launchers: []Launcher{
			{ID: "shell", Label: "Shell", Command: nil, Description: "Default shell"},
		},
	}

	if gmuxrPath == "" {
		log.Printf("launchers: gmuxr not found, only shell available")
		return cfg
	}

	out, err := exec.Command(gmuxrPath, "adapters").Output()
	if err != nil {
		log.Printf("launchers: gmuxr adapters failed: %v", err)
		return cfg
	}

	var adapters []Launcher
	if err := json.Unmarshal(out, &adapters); err != nil {
		log.Printf("launchers: failed to parse adapter list: %v", err)
		return cfg
	}

	cfg.Launchers = append(cfg.Launchers, adapters...)
	log.Printf("launchers: discovered %d adapter(s): %v", len(adapters), adapterNames(adapters))
	return cfg
}

// resolveGmuxr finds the gmuxr binary.
// Priority: sibling to this binary > PATH lookup.
// Both gmuxd and gmuxr are always installed to the same directory.
func resolveGmuxr() string {
	if exe, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(exe), "gmuxr")
		if _, err := os.Stat(sibling); err == nil {
			return sibling
		}
	}
	if p, err := exec.LookPath("gmuxr"); err == nil {
		return p
	}
	return ""
}

func adapterNames(ls []Launcher) []string {
	names := make([]string, len(ls))
	for i, l := range ls {
		names[i] = l.ID
	}
	return names
}

func main() {
	gmuxrBin := resolveGmuxr() // resolve once, use everywhere
	if gmuxrBin != "" {
		log.Printf("gmuxr: %s", gmuxrBin)
	}
	launchConfig := discoverLaunchers(gmuxrBin)

	sessions := store.New()
	subs := discovery.NewSubscriptions(sessions)

	// Start socket-based discovery (scans /tmp/gmux-sessions/*.sock)
	// Discovery also subscribes to each runner's /events SSE for live updates.
	stopDiscovery := make(chan struct{})
	go discovery.Watch(sessions, subs, 3*time.Second, stopDiscovery)
	defer close(stopDiscovery)

	mux := http.NewServeMux()

	// ── Health + Capabilities ──

	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ok": true,
			"data": map[string]any{
				"service": "gmuxd",
				"node_id": "node-local",
				"status":  "ready",
			},
		})
	})

	mux.HandleFunc("/v1/capabilities", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ok": true,
			"data": map[string]any{
				"adapters": []string{"pi", "shell"},
				"transport": map[string]any{
					"kind":   "websocket",
					"replay": true,
				},
			},
		})
	})

	// ── Config ──

	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, r *http.Request) {
		cfg := launchConfig
		writeJSON(w, map[string]any{"ok": true, "data": cfg})
	})

	// ── Sessions ──

	mux.HandleFunc("GET /v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "data": sessions.List()})
	})

	// ── Registration (fast path for gmux-run) ──

	mux.HandleFunc("POST /v1/register", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "read error")
			return
		}

		var req struct {
			SessionID  string `json:"session_id"`
			SocketPath string `json:"socket_path"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
			return
		}

		if req.SessionID == "" || req.SocketPath == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "session_id and socket_path required")
			return
		}

		log.Printf("register: %s at %s", req.SessionID, req.SocketPath)
		if err := discovery.Register(sessions, subs, req.SocketPath); err != nil {
			log.Printf("register: failed to query meta for %s: %v", req.SessionID, err)
			writeError(w, http.StatusBadGateway, "runner_unreachable", err.Error())
			return
		}

		writeJSON(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("POST /v1/deregister", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "read error")
			return
		}

		var req struct {
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
			return
		}

		// Don't remove from store — the exit event from the subscription
		// already marked it alive: false. Just clean up the subscription.
		subs.Unsubscribe(req.SessionID)
		log.Printf("deregister: %s", req.SessionID)
		writeJSON(w, map[string]any{"ok": true})
	})

	// ── Launch ──

	mux.HandleFunc("POST /v1/launch", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "read error")
			return
		}

		var req struct {
			Cwd        string   `json:"cwd"`
			Command    []string `json:"command"`
			LauncherID string   `json:"launcher_id"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
			return
		}

		// Resolve command from launcher_id if no explicit command
		if len(req.Command) == 0 && req.LauncherID != "" {
			cfg := launchConfig
			for _, l := range cfg.Launchers {
				if l.ID == req.LauncherID {
					req.Command = l.Command
					break
				}
			}
		}

		// Empty/nil command means "shell" — use user's $SHELL
		if len(req.Command) == 0 {
			shell := os.Getenv("SHELL")
			if shell == "" {
				shell = "/bin/sh"
			}
			req.Command = []string{shell}
		}

		cwd := req.Cwd
		if cwd == "" {
			cwd = os.Getenv("HOME")
		}

		if gmuxrBin == "" {
			writeError(w, http.StatusInternalServerError, "gmuxr_not_found", "gmuxr not found (install gmuxr alongside gmuxd)")
			return
		}

		// Build args: gmuxr [--cwd dir] -- <command...>
		args := []string{}
		args = append(args, "--cwd", cwd)
		args = append(args, "--")
		args = append(args, req.Command...)

		cmd := exec.Command(gmuxrBin, args...)
		cmd.Dir = cwd
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // detach from gmuxd
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil

		if err := cmd.Start(); err != nil {
			log.Printf("launch: failed to start gmuxr: %v", err)
			writeError(w, http.StatusInternalServerError, "launch_failed", err.Error())
			return
		}

		// Don't wait — gmuxr is a detached daemon. Release the process.
		go cmd.Wait()

		log.Printf("launch: started gmuxr pid=%d cwd=%s cmd=%v", cmd.Process.Pid, cwd, req.Command)
		writeJSON(w, map[string]any{
			"ok":   true,
			"data": map[string]any{"pid": cmd.Process.Pid},
		})
	})

	// ── Session Actions ──

	mux.HandleFunc("/v1/sessions/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 {
			http.NotFound(w, r)
			return
		}
		sessionID := parts[2]
		action := ""
		if len(parts) == 4 {
			action = parts[3]
		}

		switch action {
		case "attach":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
				return
			}
			sess, ok := sessions.Get(sessionID)
			if !ok {
				writeError(w, http.StatusNotFound, "not_found", "session not found")
				return
			}
			writeJSON(w, map[string]any{
				"ok": true,
				"data": map[string]any{
					"transport":   "websocket",
					"ws_path":     "/ws/" + sessionID,
					"socket_path": sess.SocketPath,
				},
			})

		case "kill":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
				return
			}
			sess, ok := sessions.Get(sessionID)
			if !ok {
				writeError(w, http.StatusNotFound, "not_found", "session not found")
				return
			}
			// Send kill to runner — it will SIGTERM the child, which triggers
			// normal exit lifecycle (exit event → subscription updates store)
			if sess.SocketPath != "" && sess.Alive {
				if err := discovery.KillSession(sess.SocketPath); err != nil {
					log.Printf("kill: %s: runner kill failed: %v", sessionID, err)
				}
			}
			writeJSON(w, map[string]any{"ok": true, "data": map[string]any{}})

		default:
			http.NotFound(w, r)
		}
	})

	// ── WebSocket proxy ──

	mux.HandleFunc("/ws/{sessionID}", wsproxy.Handler(func(sessionID string) (string, error) {
		sess, ok := sessions.Get(sessionID)
		if !ok {
			return "", fmt.Errorf("session %s not found", sessionID)
		}
		if sess.SocketPath == "" {
			return "", fmt.Errorf("session %s has no socket", sessionID)
		}
		return sess.SocketPath, nil
	}))

	// ── SSE Events ──

	mux.HandleFunc("GET /v1/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Send current state as upserts
		for _, sess := range sessions.List() {
			s := sess
			sendSSE(w, "session-upsert", store.Event{
				Type:    "session-upsert",
				ID:      s.ID,
				Session: &s,
			})
		}
		flusher.Flush()

		// Stream updates
		ch, cancel := sessions.Subscribe()
		defer cancel()

		notify := r.Context().Done()
		for {
			select {
			case <-notify:
				return
			case ev, open := <-ch:
				if !open {
					return
				}
				sendSSE(w, ev.Type, ev)
				flusher.Flush()
			}
		}
	})

	addr := envOr("GMUXD_ADDR", ":8790")
	log.Printf("gmuxd listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func sendSSE(w http.ResponseWriter, event string, payload any) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, bytes)
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":    false,
		"error": map[string]any{"code": code, "message": message},
	})
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
