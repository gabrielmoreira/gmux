// Package tsauth provides an optional tailscale (tsnet) HTTPS listener
// with identity-based access control.
//
// When enabled, gmuxd joins the user's tailnet and serves the same HTTP
// handler as the localhost listener, but wrapped in middleware that:
//  1. Enforces HTTPS (tsnet provides automatic Let's Encrypt certs).
//  2. Checks the connecting peer's tailscale identity (via WhoIs) against
//     a configured allow list of login names and/or device names.
//
// The allow list is fail-closed: if empty, all connections are rejected.
package tsauth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

// Config mirrors the tailscale section of the gmuxd config file.
type Config struct {
	Hostname string
	Allow    []string // login names ("user@github") or device names ("my-phone")
}

// Listener manages a tsnet server and its HTTPS listener.
type Listener struct {
	srv *tsnet.Server
	lc  *tailscale.LocalClient
	cfg Config
}

// Start joins the tailnet and begins serving handler over HTTPS on :443.
// It blocks in a goroutine — call Shutdown to stop.
func Start(cfg Config, stateDir string, handler http.Handler) (*Listener, error) {
	if len(cfg.Allow) == 0 {
		return nil, fmt.Errorf("tsauth: tailscale.allow is empty — no one would be able to connect (fail-closed)")
	}

	srv := &tsnet.Server{
		Hostname: cfg.Hostname,
		Dir:      filepath.Join(stateDir, "tsnet"),
	}

	// Start the tsnet node and wait for it to be ready.
	if err := srv.Start(); err != nil {
		return nil, fmt.Errorf("tsauth: tsnet start: %w", err)
	}

	lc, err := srv.LocalClient()
	if err != nil {
		srv.Close()
		return nil, fmt.Errorf("tsauth: local client: %w", err)
	}

	l := &Listener{
		srv: srv,
		lc:  lc,
		cfg: cfg,
	}

	// HTTPS listener with automatic certs from tailscale.
	ln, err := srv.ListenTLS("tcp", ":443")
	if err != nil {
		srv.Close()
		return nil, fmt.Errorf("tsauth: listen TLS: %w", err)
	}

	go func() {
		authed := l.authMiddleware(handler)
		if err := http.Serve(ln, authed); err != nil {
			log.Printf("tsauth: serve: %v", err)
		}
	}()

	log.Printf("tsauth: listening on https://%s (allowed: %v)", cfg.Hostname, cfg.Allow)
	return l, nil
}

// Shutdown stops the tsnet server.
func (l *Listener) Shutdown() {
	l.srv.Close()
}

// authMiddleware wraps a handler with tailscale identity checks.
func (l *Listener) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		who, err := l.lc.WhoIs(r.Context(), r.RemoteAddr)
		if err != nil {
			log.Printf("tsauth: WhoIs(%s): %v", r.RemoteAddr, err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		loginName := who.UserProfile.LoginName
		nodeName := who.Node.ComputedName

		if !l.isAllowed(loginName, nodeName) {
			log.Printf("tsauth: DENIED %s (login=%s device=%s)", r.RemoteAddr, loginName, nodeName)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isAllowed checks if the connecting peer matches any entry in the allow list.
// Matches against login name (e.g. "user@github") or device name (e.g. "my-phone").
// Comparison is case-insensitive.
func (l *Listener) isAllowed(loginName, nodeName string) bool {
	loginLower := strings.ToLower(loginName)
	nodeLower := strings.ToLower(nodeName)

	for _, entry := range l.cfg.Allow {
		entryLower := strings.ToLower(entry)
		if entryLower == loginLower || entryLower == nodeLower {
			return true
		}
	}
	return false
}

// WaitReady blocks until the tsnet server has a tailscale IP, or ctx is cancelled.
func (l *Listener) WaitReady(ctx context.Context) error {
	deadline := time.After(30 * time.Second)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("tsauth: timed out waiting for tailscale IP")
		case <-tick.C:
			status, err := l.lc.Status(ctx)
			if err != nil {
				continue
			}
			if status.Self != nil && len(status.Self.TailscaleIPs) > 0 {
				log.Printf("tsauth: tailscale IP: %s", status.Self.TailscaleIPs[0])
				return nil
			}
		}
	}
}
