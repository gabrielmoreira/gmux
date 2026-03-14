---
title: Architecture
description: How gmuxr, gmuxd, and gmux-web connect.
---

gmux has three components. Two are Go binaries, one is a browser app.

## Components

### gmuxr — session runner

`gmuxr` wraps any command in a managed session. For each session it:

1. Allocates a PTY (pseudo-terminal)
2. Starts a WebSocket server on a Unix socket
3. Runs an **adapter** that monitors the child process
4. Exposes a status API on `$GMUX_SOCKET`

gmuxr is the **source of truth** for session state. If gmuxd restarts, it rediscovers running gmuxr instances and rebuilds its cache.

### gmuxd — machine daemon

`gmuxd` runs once per machine. It:

1. Discovers gmuxr sessions via their Unix sockets
2. Caches session state for fast sidebar rendering
3. Proxies WebSocket connections from the browser to gmuxr
4. Pushes real-time state updates to the browser via SSE
5. Runs **probes** that observe working directories

gmuxd is stateless — restart it anytime and it rebuilds from what's running.

### gmux-web — browser UI

The browser UI provides:

- A **sidebar** that groups sessions by working directory
- A **terminal** powered by xterm.js (same as VS Code)
- A **header bar** with session metadata and actions
- **Real-time updates** via Server-Sent Events

## Data flow

```
gmuxr ──Unix socket──→ gmuxd ──HTTP/SSE──→ browser
                              ──WS proxy──→ browser
```

1. gmuxr registers with gmuxd over a Unix socket
2. gmuxd pushes state changes to the browser via SSE
3. When you click a session, the browser opens a WebSocket through gmuxd to gmuxr
4. Terminal I/O flows over WebSocket; status updates flow over SSE

## Design principles

- **Runner-authoritative**: gmuxr owns session state. gmuxd is a rebuildable cache.
- **No external dependencies**: No tmux, no screen, no abduco. Just the two binaries.
- **Web-first**: Works on any device with a browser. No Electron.
- **Zero config**: No project files, no registration. Sessions group automatically.
