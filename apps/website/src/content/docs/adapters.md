---
title: Adapters
description: How gmux understands what your process is doing.
---

Adapters are session-level intelligence. They teach gmuxr how to interpret specific tools, providing rich status information that appears in the sidebar.

## How adapters work

When you run `gmuxr pi`, gmuxr recognizes the `pi` command and activates the pi adapter. The adapter monitors the child process and reports structured status:

- **thinking** — the agent is processing
- **waiting for input** — the agent needs your attention
- **editing** — the agent is modifying files

This status appears in the sidebar, so you can see at a glance which sessions need attention.

## Built-in adapters

| Adapter | Detects | Status examples |
|---------|---------|-----------------|
| **pi** | `pi` command | thinking, waiting for input, editing |
| **pytest** | `pytest` command | 47/47 passing, 3 failing, running |
| **generic** | Everything else | active, idle (30s), exited (0) |

## The generic adapter

Unknown commands get the generic adapter automatically. It provides:

- **Activity detection** — tracks stdout/stderr activity
- **Silence timeout** — reports idle after 30 seconds of no output
- **Exit tracking** — reports exit code when the process ends

## Self-reporting via `$GMUX_SOCKET`

Any child process can report its own status without a custom adapter. gmuxr sets the `$GMUX_SOCKET` environment variable pointing to its Unix socket:

```bash
curl -X PUT --unix-socket "$GMUX_SOCKET" \
  http://localhost/status \
  -d '{"state": "building", "detail": "step 3/5"}'
```

This is how tools can integrate with gmux without any changes to gmuxr itself.
