---
title: Quick Start
description: Install gmux and launch your first session in under a minute.
---

## Install

```bash
brew install gmuxapp/tap/gmux
```

Or download both binaries (`gmuxd` + `gmuxr`) from [GitHub Releases](https://github.com/gmuxapp/gmux/releases).

## Run

```bash
gmuxr pi                    # launch a coding agent
gmuxr -- pytest --watch     # or any command after --
gmuxr -- make build
```

Open **[localhost:8790](http://localhost:8790)**. Sessions appear in the sidebar grouped by project directory. Click one to attach a live terminal.

`gmuxr` auto-starts the daemon (`gmuxd`) on first run — there's nothing else to set up.

## App mode

Run gmux as a standalone window instead of a browser tab:

```bash
google-chrome --app=http://localhost:8790
# macOS:
open -na "Google Chrome" --args --app=http://localhost:8790
```

You can also install it as a PWA from the browser menu (⋮ → Install gmux).

App mode matters for keyboard shortcuts — browsers reserve `Ctrl+T`, `Ctrl+N`, `Ctrl+W` and others. In a tab, these control the browser. In app mode, they pass through to your terminal.

## Mobile

Open the same URL on your phone. The bottom bar provides `esc`, `tab`, `ctrl`, arrow keys, and `enter`.

## Architecture

```
gmuxr (per session)  →  gmuxd (per machine)  →  browser
  PTY + WebSocket         discovery + proxy        sidebar + terminal
```

See [Architecture](/architecture) for details, or [CONTRIBUTING.md](https://github.com/gmuxapp/gmux/blob/main/CONTRIBUTING.md) for the development setup.
