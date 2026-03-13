// Package wsproxy provides WebSocket reverse proxy from gmuxd to gmux-run
// session sockets. Browser connects to gmuxd /ws/{session_id}, gmuxd proxies
// bidirectionally to the gmux-run Unix socket for that session.
package wsproxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
)

// SocketResolver maps a session ID to a Unix socket path.
type SocketResolver func(sessionID string) (string, error)

// Handler returns an http.HandlerFunc that proxies WebSocket connections.
// sessionID must be extracted by the caller and passed via the request context or URL.
func Handler(resolve SocketResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.PathValue("sessionID")
		if sessionID == "" {
			http.Error(w, "missing session_id", http.StatusBadRequest)
			return
		}

		sockPath, err := resolve(sessionID)
		if err != nil {
			log.Printf("wsproxy: resolve %s: %v", sessionID, err)
			http.Error(w, fmt.Sprintf("session not found: %v", err), http.StatusNotFound)
			return
		}

		// Accept browser WebSocket
		clientConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true, // CORS handled at gmuxd level
		})
		if err != nil {
			log.Printf("wsproxy: accept: %v", err)
			return
		}
		defer clientConn.Close(websocket.StatusNormalClosure, "")

		// Connect to gmux-run's Unix socket
		ctx := r.Context()
		backendConn, _, err := websocket.Dial(ctx, "ws://localhost/", &websocket.DialOptions{
			HTTPClient: &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
						return net.Dial("unix", sockPath)
					},
				},
			},
		})
		if err != nil {
			log.Printf("wsproxy: dial backend %s: %v", sockPath, err)
			clientConn.Close(websocket.StatusInternalError, "backend unavailable")
			return
		}
		defer backendConn.Close(websocket.StatusNormalClosure, "")

		log.Printf("wsproxy: proxying %s via %s", sessionID, sockPath)

		// Bidirectional proxy
		var wg sync.WaitGroup
		wg.Add(2)

		// Backend → Client (PTY output)
		go func() {
			defer wg.Done()
			proxyFrames(ctx, backendConn, clientConn, "backend→client")
		}()

		// Client → Backend (keyboard input + resize)
		go func() {
			defer wg.Done()
			proxyFrames(ctx, clientConn, backendConn, "client→backend")
		}()

		wg.Wait()
		log.Printf("wsproxy: session %s disconnected", sessionID)
	}
}

func proxyFrames(ctx context.Context, src, dst *websocket.Conn, label string) {
	for {
		typ, reader, err := src.Reader(ctx)
		if err != nil {
			return
		}

		writer, err := dst.Writer(ctx, typ)
		if err != nil {
			return
		}

		if _, err := io.Copy(writer, reader); err != nil {
			return
		}

		if err := writer.Close(); err != nil {
			return
		}
	}
}
