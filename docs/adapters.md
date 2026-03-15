# Adapters

Adapters teach gmux how to work with specific tools. When gmuxr launches
a process, the adapter recognizes the command, optionally modifies the
launch, and monitors PTY output to report status to the sidebar.

## Where adapters live

Adapter definitions live in a **shared package** (`packages/adapter`)
imported by both gmuxr and gmuxd. Each adapter is a single struct that
implements the base interface plus any capability interfaces it supports.

The two components use different parts of the same adapter:

| Concern | Component | Interface |
|---------|-----------|-----------|
| Command matching | **gmuxr** | `Adapter.Match()` |
| Environment injection | **gmuxr** | `Adapter.Env()` |
| PTY output monitoring | **gmuxr** | `Adapter.Monitor()` |
| Launcher registration | **both** | `Launcher` struct via `init()` |
| Session file discovery | **gmuxd** | `SessionFiler.SessionDir()` |
| Session file parsing | **gmuxd** | `SessionFiler.ParseSessionFile()` |
| Live file monitoring | **gmuxd** | `FileMonitor.ParseNewLines()` |
| Resume command | **gmuxd** | `Resumer.ResumeCommand()` |
| Session file attribution | **gmuxd** | `SessionFiler` + content similarity |
| Resumable discovery | **gmuxd** | `SessionFiler` + `Resumer` |

The split exists because gmuxr is per-session (sees one process's PTY)
while gmuxd is per-machine (sees all files, all sessions). File-level
concerns require the global view.

## Interfaces

Adapters are defined through a base interface plus opt-in capability
interfaces. The base is required; capabilities are checked via type
assertion (`if sf, ok := a.(SessionFiler); ok { ... }`).

### Base interface (required, all adapters)

```go
type Adapter interface {
    Name() string
    Match(command []string) bool
    Env(ctx EnvContext) []string
    Monitor(output []byte) *Status
}
```

**Name** — adapter identifier: `"shell"`, `"pi"`, etc. Used in session
metadata (`kind` field) and for `GMUX_ADAPTER` env override matching.

**Match** — called once at launch. Receives the full command array.
Should be cheap — match on binary base name and argument patterns:

```go
func (p *Pi) Match(cmd []string) bool {
    for _, arg := range cmd {
        base := filepath.Base(arg)
        if base == "pi" || base == "pi-coding-agent" {
            return true
        }
        if arg == "--" { break }
    }
    return false
}
```

The cost of a false negative is low — the shell adapter catches
everything, and `GMUX_ADAPTER=pi` overrides matching entirely.

**Env** — returns adapter-specific environment variables. The runner
automatically sets `GMUX`, `GMUX_SOCKET`, `GMUX_SESSION_ID`,
`GMUX_ADAPTER`, and `GMUX_VERSION`. Most adapters return nil.

**The command is never modified by adapters.** What the user (or
launcher) specified is exactly what runs. This ensures session metadata
matches reality, resume doesn't need original-vs-prepared disambiguation,
and there's no surprising flag injection.

**Monitor** — called on **every PTY read** with raw bytes. Must be very
cheap. Returns nil when there's nothing to report. When it returns a
`Status`, the runner propagates it via SSE to the sidebar.

### SessionFiler — session file discovery and parsing

```go
type SessionFiler interface {
    SessionDir(cwd string) string
    ParseSessionFile(path string) (*SessionFileInfo, error)
}
```

Implemented by adapters whose tools write session files to disk. Used
by **gmuxd** for:
- **Resumable discovery** — scan `SessionDir()`, parse each file
- **Attribution** — after matching a file to a session, extract resume_key
- **Title extraction** — first user message or explicit name

**SessionDir** returns where the tool stores files for a given cwd.
Pi: `~/.pi/agent/sessions/--<encoded-cwd>--/`.

**ParseSessionFile** reads a file and returns display metadata:
ID (becomes resume_key), title, cwd, created time, message count.

### FileMonitor — live file event extraction

```go
type FileMonitor interface {
    ParseNewLines(lines []string) []FileEvent
}
```

Implemented by adapters that want to react to changes in their
attributed session file. Called by **gmuxd** when inotify fires on an
attributed file.

For pi: parses new JSONL lines, looks for `session_info` entries
(name changes via `/name` command), message count updates, etc.

### Resumer — session resume support

```go
type Resumer interface {
    ResumeCommand(info *SessionFileInfo) []string
    CanResume(path string) bool
}
```

Implemented by adapters whose sessions can be resumed after exit. Used
by **gmuxd** to:
- Filter which files are actually resumable (valid, non-empty)
- Generate the command for the UI's resume action

For pi: `ResumeCommand` returns `["pi", "--session", path, "-c"]`.
`CanResume` checks for valid header + at least one message.

### Capability composition

| Adapter | Base | SessionFiler | FileMonitor | Resumer |
|---------|------|-------------|-------------|---------|
| Shell | ✓ | — | — | — |
| Pi | ✓ | ✓ | ✓ | ✓ |
| (future opencode) | ✓ | ✓ | — | ✓ |
| (future pytest) | ✓ | — | — | — |

Shell is the simplest — just OSC title parsing. Pi is the richest.
New adapters implement only what they support.

## Status

```go
type Status struct {
    Label string  // Short text: "working", "3/10 passed"
    State string  // Visual treatment: active|attention|success|error|paused|info
    Icon  string  // Optional icon hint (emoji)
    Title string  // If set, updates the session's display title
}
```

### Status states

| State | Meaning | Sidebar indicator |
|-------|---------|-------------------|
| `active` | Working, processing | Green pulsing dot |
| `attention` | Needs user input | Orange/amber dot |
| `success` | Completed successfully | Green check |
| `error` | Something went wrong | Red dot |
| `paused` | Idle but resumable | Grey dot |
| `info` | Informational | Blue dot |

A session with no reported status shows a dim green dot (alive, quiet).

### Title updates

If `Status.Title` is set, it replaces the session's display title in the
sidebar. The shell adapter uses this to propagate OSC 0/2 terminal title
sequences (e.g., fish/zsh set the title on directory change).

## Adapter resolution

When gmuxr launches a command:

1. **`GMUX_ADAPTER` env override** — if set, use that adapter directly.
   Escape hatch for wrappers and aliases where binary name matching fails.
2. **Walk registered adapters** — call `Match()` in registration order.
   First match wins.
3. **Shell fallback** — always matches, always last.

## Built-in adapters

### Shell (fallback)

- **Capabilities**: `Adapter` only
- **Matches**: everything (catch-all)
- **Monitor**: parses OSC 0/2 title sequences for live sidebar titles
- **Status**: none — shells don't report activity states
- **Launcher**: always added by gmuxd as default; not in `Launchers` slice

### Pi (coding agent)

- **Capabilities**: `Adapter` + `SessionFiler` + `FileMonitor` + `Resumer`
- **Matches**: `pi` or `pi-coding-agent` as base name in any arg position
- **Monitor**: detects braille spinner + "Working..." → `active` status
- **Launcher**: `{id: "pi", label: "pi", command: ["pi"]}`
- **SessionDir**: `~/.pi/agent/sessions/--<cwd-encoded>--/`
- **ParseSessionFile**: reads JSONL, extracts UUID, title
  (session_info.name > first user message > "(new)"), message count
- **ParseNewLines**: watches for `session_info` name changes → title update
- **ResumeCommand**: `["pi", "--session", "<path>", "-c"]`
- **CanResume**: valid header + at least one message

## Adding an adapter

One file per adapter in `packages/adapter/adapters/`. The file registers
itself via `init()`:

```go
package adapters

func init() {
    Launchers = append(Launchers, Launcher{
        ID: "myapp", Label: "MyApp", Command: []string{"myapp"},
    })
    All = append(All, NewMyApp())
}

type MyApp struct{}

func NewMyApp() *MyApp { return &MyApp{} }
func (m *MyApp) Name() string { return "myapp" }
func (m *MyApp) Match(cmd []string) bool { /* ... */ }
func (m *MyApp) Env(ctx adapter.EnvContext) []string { return nil }
func (m *MyApp) Monitor(output []byte) *adapter.Status { /* ... */ }
```

To add session file support, implement `SessionFiler` on the same struct:

```go
func (m *MyApp) SessionDir(cwd string) string {
    return filepath.Join(os.UserHomeDir(), ".myapp/sessions", encode(cwd))
}

func (m *MyApp) ParseSessionFile(path string) (*adapter.SessionFileInfo, error) {
    // Read file, extract title/ID/metadata
}
```

To add live file monitoring, implement `FileMonitor`:

```go
func (m *MyApp) ParseNewLines(lines []string) []adapter.FileEvent {
    // Parse new lines, return title/status changes
}
```

To add resume support, implement `Resumer`:

```go
func (m *MyApp) ResumeCommand(info *adapter.SessionFileInfo) []string {
    return []string{"myapp", "resume", info.ID}
}

func (m *MyApp) CanResume(path string) bool {
    // Check if file represents a valid resumable session
}
```

No other files need to be touched. Components discover capabilities
via type assertion at runtime.

## gmuxd responsibilities

gmuxd uses the adapter's capability interfaces to provide file-level
intelligence. It imports the same shared adapter package as gmuxr.

### Launcher discovery

At startup, gmuxd queries `Launchers` from the adapter registry.
Prepends shell as the default and serves via `GET /v1/config`.

### Session file attribution (ADR-0009)

For each unique cwd with live sessions, gmuxd checks if any adapter
implements `SessionFiler`. If so, it watches `SessionDir(cwd)` with
inotify. When a file is written:

1. **Single session in cwd** → trivially attribute
2. **Multiple sessions** → content similarity match (ADR-0009)

Attribution sets `resume_key` on the session. Sticky — only re-checks
when a different file starts receiving writes.

### Live file monitoring

After attribution, gmuxd keeps watching the file. On new writes:

1. Read new lines (tracked offset per file)
2. If adapter implements `FileMonitor`, call `ParseNewLines(newLines)`
3. Apply returned `FileEvent`s to the session store (title changes, etc.)

This is how `/name` in pi updates the sidebar — gmuxd sees the new
`session_info` JSONL line, the adapter's `ParseNewLines` extracts the
title, gmuxd updates the store, SSE pushes it to the frontend.

### Resumable session discovery

For each registered adapter implementing `SessionFiler` + `Resumer`:

1. List files in `SessionDir()` for known cwds
2. Filter with `CanResume(path)`
3. Parse with `ParseSessionFile(path)`
4. Deduplicate against live sessions by `resume_key`
5. Push as `session-upsert` with `resumable: true` via SSE

## Session states from the UI's perspective

| State | Indicator | Terminal | Close button | Source |
|-------|-----------|----------|--------------|--------|
| Alive, no status | Dim green dot | Interactive | × or − | Process running |
| Alive, active | Green pulsing | Interactive | × or − | Monitor() |
| Alive, attention | Orange dot | Interactive | × or − | Monitor() or child PUT /status |
| Alive, error | Red dot | Interactive | × or − | Monitor() or child PUT /status |
| Resumable | Hollow dot | None | × (forget) | gmuxd scan |
| Gone | Not shown | None | — | Dismissed or exited (no resume) |

The close button shape depends on whether the adapter supports resume:
- **×** — session will be forgotten (shell, or dismissing a resumable entry)
- **−** — session will be closed but entry remains as resumable (pi)

Since no adapter implements resume yet, all sessions show × today.
When pi gets resume support, its sessions automatically show −.

See [ADR-0007](adr/0007-session-lifecycle-and-close-semantics.md).

## Child awareness protocol

Any process running inside gmuxr can detect and communicate with gmux
through environment variables set automatically by the runner:

| Variable | Value | Purpose |
|----------|-------|---------|
| `GMUX` | `1` | Detection flag |
| `GMUX_SOCKET` | `/tmp/gmux-sessions/sess-abc.sock` | Communication channel |
| `GMUX_SESSION_ID` | `sess-abc` | Session identity |
| `GMUX_ADAPTER` | `pi` | Which adapter matched |
| `GMUX_VERSION` | `0.1.0` | Protocol version |

Children can optionally call back to the runner:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/status` | PUT | Set application status (highest priority) |
| `/meta` | PATCH | Update title, subtitle |

Priority: child self-report > gmuxd FileMonitor > adapter Monitor() > process defaults.

## Testing

**Unit tests** run with each component's test suite — adapter matching,
parsing, status extraction.

**Integration tests** live alongside the adapter code with a
`//go:build integration` tag:

```bash
go test -tags integration -v -timeout 120s -run TestPi \
  ./packages/adapter/adapters/
```

These launch real processes through PTYs and verify adapter behavior.
They require the target binaries to be installed (e.g., `pi`) and are
skipped if not found. Tests document observable behavior patterns (output
timing, file creation lifecycle) that the attribution and monitoring
logic depends on.
