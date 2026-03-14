---
title: Introduction
description: What gmux is and why it exists.
---

gmux is a session manager for AI agents, test runners, and long-running processes. It gives you a live, interactive terminal for each one — grouped by project, with real-time status pushed to your browser.

## The problem

You're running three AI agents, a test watcher, and a build process across two projects. They're scattered across tmux panes, terminal tabs, and background jobs. When an agent needs input, you don't notice for ten minutes. When tests fail, you find out when you context-switch back.

## The solution

gmux wraps each command in a **managed session** with:

- A full terminal (xterm.js — the same engine as VS Code)
- Real-time status that tells you what the process is doing
- Automatic grouping by project directory
- A browser UI that works on desktop and phone

No Electron, no desktop app, no tmux. Just two small Go binaries and a browser tab.

## Key concepts

### Sessions

A session is any command launched through `gmuxr`. It gets a PTY, a WebSocket for terminal access, and an adapter that monitors what the child process is doing.

```bash
gmuxr pi                    # launch a coding agent
gmuxr -- pytest --watch     # launch a test watcher
gmuxr -- make build         # any command works
```

### Adapters

Adapters are session-level intelligence. They teach gmux how to interpret specific tools — when pi is thinking vs waiting for input, when pytest has failures, when a build is done. Unknown commands get generic activity tracking.

### Probes

Probes are directory-level intelligence. They observe working directories and report context like git branch, dirty state, and open PRs. They enrich the sidebar's folder headings.

### Architecture

```
gmuxr (per session) → gmuxd (per machine) → browser
```

**gmuxr** manages individual sessions. **gmuxd** discovers and aggregates them. The **browser** displays everything.

## Next steps

→ [Quick Start](/quick-start) — install and run gmux in 30 seconds
