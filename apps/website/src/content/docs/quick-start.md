---
title: Quick Start
description: Install gmux and launch your first session in 30 seconds.
---

## Install

```bash
go install github.com/gmuxapp/gmux/services/gmuxd@latest
go install github.com/gmuxapp/gmux/cli/gmuxr@latest
```

## Start the daemon

```bash
gmuxd &
```

gmuxd runs once per machine. It discovers sessions, caches their state, and serves the browser UI.

## Launch sessions

```bash
gmuxr pi                        # launch a coding agent
gmuxr -- pytest tests/ --watch  # launch a test watcher
gmuxr -- make build             # or any command
```

Each `gmuxr` call wraps a command in a managed session with a PTY, WebSocket, and status adapter.

## Open the UI

```bash
open http://localhost:5173
```

All three sessions appear in the sidebar, grouped by working directory, with live status indicators. Click one to attach a full terminal.

## What you'll see

Sessions from the same directory are grouped into **folders**. Each folder shows git branch and dirty state (via probes). Each session shows what the process is doing (via adapters).

The terminal is xterm.js with 128KB of scrollback that replays instantly on reconnect. Switch away and come back — nothing is lost.

## Next steps

- [Architecture](/architecture) — understand gmuxr, gmuxd, and how they connect
- [Adapters](/adapters) — how gmux understands what your process is doing
- [Probes](/probes) — how gmux enriches folder headings with project context
