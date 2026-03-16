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

## Next steps

- [Using the UI](/using-the-ui) — what the dots mean, keyboard shortcuts, mobile, launching from the browser
- [Remote Access](/remote-access) — access gmux from your phone or another machine
- [Configuration](/configuration) — config file, environment variables, file paths
- [Architecture](/architecture) — how the pieces fit together
