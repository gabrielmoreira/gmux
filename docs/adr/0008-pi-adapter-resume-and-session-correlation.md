# ADR-0008: Pi adapter resume and session correlation

- Status: Exploring
- Date: 2026-03-15

## Context

Pi stores session history as JSONL files under
`~/.pi/agent/sessions/--<cwd-encoded>--/<timestamp>_<uuid>.jsonl`.
Sessions are resumable via `pi --session <path> -c`. For gmux to show
resumable sessions in the sidebar and deduplicate them against live
sessions, we need to:

1. **Discover** resumable sessions from disk (scan JSONL files)
2. **Correlate** a running pi process with its session file
3. **Deduplicate** live sessions against resumable entries in the UI

### Challenges

- Pi does **not** hold the session file open — it opens, appends, closes
  on each write. `/proc/<pid>/fd` scanning is unreliable.
- Pi does **not** create the file on startup — only on first user
  interaction. File creation can be minutes after process start.
- Pi can **switch sessions** mid-process via the `/resume` command.
  The resume key must be updateable.
- We should **not modify pi's command line** (injecting `--session`)
  because false adapter matches would cause breakage, and the user may
  pass their own `--continue`, `--resume`, or `--session` flags.
- Pi could report its session file cooperatively (via `GMUX_SOCKET`),
  but we want a solution that works without pi changes first.
- **Multiple pi instances** in the same cwd must not confuse each other.

## Decision

### Resume key model

Sessions have an optional `resume_key` field in their metadata:

```
id:          "sess-abc123"      ← gmux-generated, for socket path
adapter:     "pi"               ← which adapter matched
resume_key:  "8afa0c0a-..."    ← optional, adapter-specific, can change
```

- The resume key is opaque to gmux core — just a string.
- For pi, it's the session UUID from the JSONL header.
- The adapter name is already in the metadata, so the key needs no prefix.
- Shell sessions never have a resume key.

### Adapter architecture split: gmuxr vs gmuxd

Not all adapter logic belongs in gmuxr. The per-session runner and the
per-machine daemon have different responsibilities:

**gmuxr (per-session, in-process):**
- `Match()` — does this command look like pi?
- `Env()` — provide adapter-specific environment variables
- `Monitor()` — parse PTY output for status/title

**gmuxd (per-machine, singleton):**
- **Session directory watching** — one inotify watcher per unique cwd,
  shared across all sessions in that folder. Not duplicated per-session.
- **Resumable session discovery** — scan JSONL files, extract metadata,
  filter dismissed entries.
- **Correlation** — match file events to live sessions using the global
  view of all PIDs, all sessions, and all file events.

This split exists because gmuxd has the **global view**: it knows all
live sessions, all PIDs, all cwds. A gmuxr sidecar only sees its own
session. Correlation requires the global view.

### Correlation strategy

The goal: when a `.jsonl` file is created or modified, determine which
live pi session (if any) is responsible.

**Signals available:**
- **inotify events** on session directories: file created/modified, with
  timestamp. inotify does not report the writer PID.
- **fanotify** (Linux 5.13+): like inotify but reports the writer PID.
  Would let us check if the writer is a descendant of a known session's
  child process (same session group via `Setsid`). Not available on macOS.
- **PTY output timing**: Monitor() sees when a conversation turn happens.
  Pi writes to the JSONL in the same event loop tick as producing output.
  Strong causal correlation.
- **PTY output content**: the JSONL file and PTY output contain the same
  conversation text. Content matching is possible but heavyweight.
- **Process start time**: known per-session, but pi doesn't write the
  file on start — only on first interaction, which could be minutes later.

**Planned approach (to be validated by exploration):**

1. gmuxd watches each active session directory with inotify
2. When a file event occurs, check which live pi sessions share that cwd
3. If only one → it's that session's file (common case)
4. If multiple → use timing correlation between file event and recent
   PTY activity from Monitor(), or fanotify PID on Linux
5. Once a session → file mapping is established, extract the UUID from
   the JSONL header and set it as the session's resume_key
6. If the file changes (via /resume), update the resume_key

**The exploration phase** (see `tests/pi-adapter/`) will instrument both
event streams to validate this approach before committing to it.

### Frontend deduplication

The frontend receives all sessions (live and resumable) through the same
SSE stream as `session-upsert` events. Resumable sessions have
`alive: false, resumable: true`.

Deduplication:
```
for each resumable entry:
  if any live session has same adapter AND same resume_key:
    hide the resumable entry
```

A live session with no resume_key yet (pi hasn't written to a file)
doesn't match anything — both entries are visible, which is correct.

### Resumable session discovery

gmuxd periodically scans pi session directories:

1. List `~/.pi/agent/sessions/*/` directories
2. For each, list `.jsonl` files, sorted by modification time
3. Read line 1 of each file: `{"type":"session","id":"<uuid>","cwd":"..."}` 
4. Optionally read forward to find first user message (for title) or
   `session_info` entry (for `/name`-set display name)
5. Filter out dismissed sessions (`~/.config/gmux/dismissed-sessions.json`)
6. Emit as resumable sessions into the store

This scan is cheap — one `readLine` per file. Capped at N most recent
per folder (default: 10), with "show more" in the UI.

### Resume command

When the user clicks a resumable session:
```
gmuxr pi --session <path> -c
```

This opens the specific session file in continue mode. Pi resumes the
conversation.

## Status

**Exploring.** The correlation strategy needs validation via instrumented
testing before we commit to implementation. Key questions:

- How tight is the timing correlation between PTY output and file write?
- Does fanotify work reliably for this use case on Linux?
- What does pi's stdout look like during startup, first interaction,
  and session switches? What patterns can Monitor() detect?
- Is the single-instance-per-cwd common case sufficient, or do we need
  multi-instance correlation from day one?

## Consequences

### Positive
- No pi changes required for basic functionality
- Gracefully upgrades if pi adds GMUX_SOCKET reporting
- Global view in gmuxd enables reliable correlation
- Single inotify watcher per cwd, not per session

### Negative
- Correlation has edge cases (multiple pi instances, same cwd, same second)
- fanotify (PID reporting) is Linux-only
- Requires adapter logic split across two binaries

### Neutral
- Resume key is optional and late-arriving — UI handles its absence
- Dismissed sessions list is a small persistent file, not a database
