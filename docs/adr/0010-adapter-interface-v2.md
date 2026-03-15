# ADR-0010: Adapter interface v2 — shared package, opt-in capabilities

- Status: Proposed
- Date: 2026-03-15

## Context

The adapter interface (ADR-0005) currently lives in
`cli/gmuxr/internal/adapter/` and only covers gmuxr's needs: Match,
Env, Monitor. But gmuxd also has adapter-specific behavior:

- **Session file discovery**: pi stores files in `~/.pi/agent/sessions/`
  with a specific directory encoding. Other tools will have different
  conventions.
- **Session file parsing**: extracting title, UUID, message count from
  pi's JSONL format. Other tools will have different formats.
- **Live file monitoring**: when an attributed file gets new writes,
  extracting meaningful events (name changes, status transitions).
- **Resume command**: `pi --session <path> -c` vs whatever another tool
  needs.
- **Resumable discovery**: listing files that represent resumable
  sessions.

This is all adapter-specific knowledge that shouldn't be hardcoded in
gmuxd. But gmuxd can't import gmuxr's internal packages.

## Decision

### Move adapter package to a shared location

```
packages/adapter/          # shared Go module
  adapter.go               # core interfaces + types
  adapters/                # built-in implementations
    shell.go
    pi.go
    launchers.go
```

Both `cli/gmuxr` and `services/gmuxd` import `packages/adapter`.
Add `./packages/adapter` to `go.work`.

### Core interface (required, all adapters)

```go
type Adapter interface {
    // Identity
    Name() string
    Match(command []string) bool

    // gmuxr hooks
    Env(ctx EnvContext) []string
    Monitor(output []byte) *Status
}
```

This is the base. Shell implements this and nothing else. Every adapter
must implement it.

### Opt-in capability interfaces

Additional interfaces that adapters implement when they support richer
behavior. Components check with type assertions:

```go
if sf, ok := adapter.(SessionFiler); ok {
    dir := sf.SessionDir(cwd)
    // ...
}
```

#### SessionFiler — session file discovery and parsing

```go
// SessionFiler is implemented by adapters whose tools write session
// files to disk (pi, claude-code, etc).
type SessionFiler interface {
    // SessionDir returns the directory where this tool stores session
    // files for the given cwd. Returns "" if unknown or not applicable.
    SessionDir(cwd string) string

    // ParseSessionFile reads a session file and returns display metadata.
    // Called by gmuxd for resumable discovery and live file monitoring.
    ParseSessionFile(path string) (*SessionFileInfo, error)
}

type SessionFileInfo struct {
    ID           string    // Tool's session identifier (becomes resume_key)
    Title        string    // Display title for sidebar
    Cwd          string    // Working directory
    Created      time.Time // When the session was created
    MessageCount int       // Number of messages/interactions
    FilePath     string    // Absolute path to the file
}
```

Called by gmuxd during:
- **Resumable discovery**: scan `SessionDir()` for files, parse each
- **Attribution**: after matching a file to a session, parse for resume_key

#### FileMonitor — live file event extraction

```go
// FileMonitor is implemented by adapters that want to react to changes
// in their attributed session file. gmuxd calls ParseNewLines when
// inotify fires on an attributed file.
type FileMonitor interface {
    // ParseNewLines receives lines appended since the last read.
    // Returns events that should update the session's state.
    // Called by gmuxd on IN_CLOSE_WRITE for the attributed file.
    ParseNewLines(lines []string) []FileEvent
}

type FileEvent struct {
    Title  string  // If non-empty, update session title
    Status *Status // If non-nil, update session status
    // Future: subtitle, metadata, etc.
}
```

Called by gmuxd when:
- An attributed file gets new writes
- Extracts meaningful changes (e.g., `session_info` name → title update)

For pi: parses new JSONL lines, looks for `session_info` entries with
name changes, possibly message role transitions for richer status.

#### Resumer — session resume support

```go
// Resumer is implemented by adapters whose sessions can be resumed
// after the process exits.
type Resumer interface {
    // ResumeCommand returns the command to resume the given session.
    ResumeCommand(info *SessionFileInfo) []string

    // CanResume returns whether a session file represents a resumable
    // session (vs a corrupted/empty/incompatible one).
    CanResume(path string) bool
}
```

Called by gmuxd when:
- User clicks a resumable session in the UI
- gmuxd needs to filter which files are actually resumable

For pi: `ResumeCommand` returns `["pi", "--session", info.FilePath, "-c"]`.
`CanResume` checks the file has a valid header and at least one message.

### What each component uses

```
gmuxr:
  Adapter.Name()          — session metadata (kind field)
  Adapter.Match()         — adapter resolution at launch
  Adapter.Env()           — child environment setup
  Adapter.Monitor()       — PTY output parsing

gmuxd:
  Adapter.Name()          — identify adapters for store/config
  Adapter.Match()         — (not used directly; launchers carry adapter name)
  SessionFiler            — resumable discovery, attribution parsing
  FileMonitor             — live file event extraction
  Resumer                 — resume command generation, filtering
```

### Capability composition

```
Shell:     Adapter
Pi:        Adapter + SessionFiler + FileMonitor + Resumer
Opencode:  Adapter + SessionFiler + Resumer  (hypothetical)
Pytest:    Adapter                            (hypothetical, no files)
```

The shell adapter is the simplest. Pi is the richest. New adapters
pick which capabilities they need.

### Interface detection pattern

```go
// In gmuxd discovery code:
for _, a := range registry.All() {
    sf, ok := a.(adapter.SessionFiler)
    if !ok {
        continue // this adapter doesn't have session files
    }
    dir := sf.SessionDir(cwd)
    if dir == "" {
        continue
    }
    // scan dir for resumable sessions...
    files := listFiles(dir)
    for _, f := range files {
        info, err := sf.ParseSessionFile(f)
        // ...
    }
}
```

No special registration, no maps, no configuration. Type assertions
on the adapter instance. If it implements the interface, the capability
is available.

## Package layout

```
packages/
  adapter/
    go.mod                    # github.com/gmuxapp/gmux/packages/adapter
    adapter.go                # Adapter interface, Status, EnvContext
    capabilities.go           # SessionFiler, FileMonitor, Resumer
    adapters/
      shell.go                # Shell: Adapter
      pi.go                   # Pi: Adapter + SessionFiler + FileMonitor + Resumer
      pi_files.go             # Pi's SessionFiler + FileMonitor implementation
      launchers.go            # Launcher type + registry
```

`pi.go` has Match/Env/Monitor (PTY-side). `pi_files.go` has
SessionDir/ParseSessionFile/ParseNewLines/ResumeCommand/CanResume
(file-side). Same struct, same package, split by concern.

## Consequences

### Positive
- Clean separation: each interface is a single concern
- Opt-in: adapters only implement what they support
- Shared: both gmuxr and gmuxd import the same package
- Discoverable: type assertion pattern is idiomatic Go
- Extensible: new capabilities = new interfaces, no breaking changes

### Negative
- Package move: adapter code moves from gmuxr internal to shared
- New Go module in workspace: minor build complexity
- Pi adapter grows: ~3 interfaces beyond base, but each is small

### Migration
1. Create `packages/adapter/` with new module
2. Move existing types + interfaces
3. Add capability interfaces
4. Move pi/shell adapter implementations
5. Update gmuxr and gmuxd imports
6. Move `ReadPiSessionInfo` → `pi_files.go` as `ParseSessionFile`
