---
title: Introduction
description: What gmux is and why it exists.
---

gmux is a browser-based session manager for AI agents, test runners, and long-running processes. It gives you a live terminal for each one, grouped by working directory, with status updates that help you notice what needs attention.

## Why it exists

Long-running command-line work is easy to start and annoying to supervise. AI agents, watchers, builds, and shells end up scattered across tabs and panes. gmux puts them in one place and makes them visible from a browser.

## What gmux does today

- launches commands as managed sessions through `gmuxr`
- keeps per-session state in the runner
- aggregates live sessions in `gmuxd`
- exposes a browser UI for browsing sessions and attaching to terminals
- uses adapters to add tool-specific status and metadata
- supports resumable sessions for tools that have file-backed adapters, such as `pi`

## Core concepts

### Sessions

A session is any command launched through `gmuxr`.

```bash
gmuxr pi
gmuxr -- make build
gmuxr -- pytest --watch
```

Each session gets a PTY, runner-owned state, and a terminal attachment path.

### Adapters

Adapters teach gmux how to interpret specific tools. The built-in adapters currently matter most for:

- **shell** — terminal title tracking
- **pi** — live status, file-backed titles, and resume

See [Adapters](/adapters) for the high-level overview.

### Runtime split

At a high level:

```text
gmuxr (per session) → gmuxd (per machine) → browser client
```

- `gmuxr` owns the child process and its live state
- `gmuxd` discovers sessions and serves aggregated machine-level state
- the browser UI renders the session list and terminal view

See [Architecture](/architecture) for the broad system view and [Adapter Architecture](/develop/adapter-architecture) for adapter-specific runtime details.

## TODO

- Document the user-facing browser/UI flow more concretely, with screenshots or a short walkthrough.
- Add a short page on session lifecycle and resume semantics from a user's point of view.

## Next steps

- [Quick Start](/quick-start)
- [Architecture](/architecture)
- [Adapters](/adapters)
