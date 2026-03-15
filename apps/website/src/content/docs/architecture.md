---
title: Architecture
description: "Broad system structure: runner, daemon, and web client."
---

This page is the high-level architecture view. It explains the major runtime pieces and how they relate.

For adapter-specific internals — including session-file discovery, attribution, resumable sessions, and child callbacks — see [Adapter Architecture](/develop/adapter-architecture).

## Core runtime pieces

### `gmuxr` — session runner

`gmuxr` runs once per session. It:

- launches the child under a PTY
- owns the live session state
- exposes the session on a Unix socket
- accepts terminal attachment and child callbacks
- runs adapter logic over the child output

`gmuxr` is the source of truth for a live session.

### `gmuxd` — machine daemon

`gmuxd` runs once per machine. It:

- discovers live runner sockets
- reads and caches session metadata
- subscribes to live updates
- exposes machine-level APIs for sessions, launchers, attach, kill, and resume
- proxies terminal websocket connections to the right runner

`gmuxd` is rebuildable. If it restarts, it can rediscover running sessions.

### Web client

The repo currently contains separate web pieces for the UI and API layer:

- `apps/gmux-web` — browser frontend
- `apps/gmux-api` — API layer used by the frontend during development

The exact production serving/deployment story is still in flux, so this page stays focused on the runtime responsibilities rather than packaging.

## Data flow

At a high level:

```text
gmuxr ──Unix socket──→ gmuxd ──HTTP/SSE/WS──→ browser client
```

Typical flow:

1. `gmuxr` launches a session and exposes it on a socket
2. `gmuxd` discovers the socket and reads session metadata
3. `gmuxd` subscribes to updates from the runner
4. the browser reads machine-level state from the daemon-facing API
5. terminal attachment is proxied back to the owning runner

## Design principles

- **runner-authoritative** — live session truth lives in `gmuxr`
- **rebuildable daemon** — `gmuxd` can recover by rediscovering runners
- **browser-first supervision** — the UI is reachable without a desktop-specific shell app
- **adapter extensibility** — tool-specific behavior is layered on top of generic process supervision

## What this page intentionally does not cover

This page does not try to explain:

- adapter capability interfaces
- session-file scanning and attribution
- child self-report protocol details
- resume mechanics for specific integrations

Those details live in [Adapter Architecture](/develop/adapter-architecture).

## TODO

- Add one concrete end-to-end example: launch command → appears in sidebar → attach terminal → resume later.
- Add a small diagram that reflects the current repo layout (`gmuxr`, `gmuxd`, `gmux-api`, `gmux-web`) instead of the older simplified 3-box view.
- Document the intended production deployment shape once that stabilizes.
