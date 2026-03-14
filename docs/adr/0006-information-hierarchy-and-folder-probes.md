# ADR-0006: Information hierarchy, folder probes, and UI surfaces

- Status: Proposed
- Date: 2026-03-14

## Context

gmux manages sessions (PTY processes with agent conversations). The sidebar groups them, the terminal shows one at a time. But sessions alone don't tell the full story of what's happening in a project — you also want to know: is the repo dirty? Is CI passing? Is there a PR open? What branch am I on?

This ADR defines the information hierarchy, introduces **folder probes** as a second extensibility layer alongside session adapters, and maps both to UI surfaces.

### Design conversation that led here

We explored several directions before converging:

**Workspace orchestration (rejected for core).** The original vision included creating isolated worktree copies, launching agents in them, tracking CI across all of them. This is valuable but it's a different product. gmux core manages sessions and displays metadata — orchestration tools (scripts, adapters) live outside and talk to gmux via its APIs.

**Project grouping (deferred).** We considered a "project" layer that groups multiple worktrees of the same repo. Determining whether two directories are "the same project" (git remote URL? naming convention? config file?) introduces ambiguity and configuration burden. For now, each working directory is its own folder. Two worktrees show as two folders: `gmux (main)` and `gmux (feature-x)`. Visual proximity and naming do the grouping work. If this proves insufficient, a project layer can be added later without breaking the model.

**cmux-style workspaces with tabs and splits (rejected for v1).** gmux runs in the browser. The browser already has tabs, windows, and tiling (especially with tiling WMs). Rebuilding a window manager inside the browser fights the host environment. Each gmux browser tab shows one view with one terminal. The browser is the workspace manager.

**The mobile handoff insight.** If views were only owned by browser tabs, switching to a phone would require reconfiguring everything. But the mobile use case is fundamentally different: not "replicate my desktop layout" but "what needs my attention right now, across everything?" The bare URL `/` serves this triage purpose naturally — all sessions, sorted by status priority. Desktop users scope via bookmarks with URL params. No server-side view state to sync.

## Decision

### Terminology

Two distinct extensibility mechanisms, clearly named for their nature:

| Term | Level | Where it runs | Nature | Example |
|------|-------|---------------|--------|---------|
| **Adapter** | Session | gmuxr (in-process, Go) | Active — manages child process lifecycle, monitors output, has side effects | pi, opencode, generic |
| **Probe** | Folder | gmuxd (script or Go) | Passive — observes a directory, reports metadata, read-only | git, github-pr, jj |

**Adapters** adapt behavior. They sit between gmuxr and the child process, intercepting launch, monitoring output, running sidecars. They're compiled into gmuxr.

**Probes** sense state. They look at a directory and report what they find. They're read-only, stateless, and can be shell scripts. A probe that observes a `.git` directory doesn't need to be a Go plugin — it runs `git status` and returns JSON.

### Information hierarchy

Two levels, both automatic:

```
Folder (working directory)
├── metadata from probes (branch, PR status, dirty state, ...)
├── aggregate status from child sessions
├── action buttons from probes + adapters
│
├── Session A (gmuxr process)
│   ├── metadata from adapter (status, title, subtitle)
│   └── action buttons from adapter
│
├── Session B
│   └── ...
│
└── Session C (e.g. test watcher)
    └── adapter reports test results → also visible at folder level
```

**Folders are discovered, not created.** When gmuxd sees sessions with the same working directory, it groups them. The folder exists because sessions exist in it. No folder CRUD, no manual organization.

**Folder state is derived from two sources:**

1. **Probes** — observe the directory itself (git status, file watchers, API queries)
2. **Session aggregation** — computed from child sessions:
   - Status dot: worst status of any child session (attention > error > active > paused > idle > dead)
   - Session count
   - Any session-contributed metadata (e.g. test runner session reports "3/47 failing" — folder can surface this)

Sessions don't explicitly "push" metadata to their folder. The folder derives aggregate state automatically. Probes add independent observations about the directory. The UI merges both.

### Folder probe interface

```go
// A Probe observes a directory and reports metadata.
// Probes are stateless and read-only. They must not modify
// the filesystem or spawn processes (beyond short-lived commands
// like `git status`).
type Probe interface {
    // Name returns the probe identifier (e.g. "git", "github-pr").
    Name() string

    // Match returns true if this probe applies to the given directory.
    // Called once when a folder is discovered. Must be fast (stat a file,
    // check for .git, etc. — no network, no subprocesses).
    Match(dir string) bool

    // Read returns the current metadata for a directory.
    // Called periodically (default: every 5s for local probes,
    // 60s for network probes). May shell out to git, gh, etc.
    // Returns nil fields to indicate "no data" (vs empty string).
    Read(dir string) (*FolderMeta, error)

    // Watch optionally returns a channel of metadata updates.
    // If non-nil, gmuxd uses this instead of polling Read().
    // The probe should use inotify/fswatch internally.
    // Return nil to use polling (simpler, good enough for most probes).
    Watch(dir string) <-chan FolderMeta
}

type FolderMeta struct {
    // Display
    Label    *string   // e.g. "main", "feature/auth"
    Tagline  *string   // e.g. "3 files changed · PR #42 open"
    Icon     *string   // optional icon/emoji hint
    Status   *string   // contributes to folder status dot: active|attention|success|error|info

    // Structured data (rendered in header bar when a session in this folder is selected)
    Fields   []Field   // key-value pairs: branch, commit, dirty file count, etc.

    // Actions (rendered as buttons in sidebar folder heading and/or header bar)
    Links    []Link    // clickable: open PR, open in editor, CI dashboard
    Actions  []Action  // executable: create branch, run tests, new session
}

type Field struct {
    Key   string  // e.g. "branch", "pr", "ci"
    Value string  // e.g. "main", "#42 (passing)", "green"
    Icon  string  // optional
}

type Link struct {
    Label string  // e.g. "PR #42"
    URL   string  // e.g. "https://github.com/org/repo/pull/42"
    Icon  string  // optional
}

type Action struct {
    Label   string   // e.g. "New pi session"
    Command []string // e.g. ["gmuxr", "--title", "new task", "pi"]
    Icon    string   // optional
    Confirm bool     // require confirmation click
}
```

### Built-in probes (phased)

**v1: git probe (local only)**
```go
func (g *GitProbe) Match(dir string) bool {
    _, err := os.Stat(filepath.Join(dir, ".git"))
    return err == nil
}

func (g *GitProbe) Read(dir string) (*FolderMeta, error) {
    branch := exec("git", "-C", dir, "symbolic-ref", "--short", "HEAD")
    status := exec("git", "-C", dir, "status", "--porcelain")
    dirty := len(strings.Split(strings.TrimSpace(status), "\n"))

    tagline := branch
    if dirty > 0 {
        tagline += fmt.Sprintf(" · %d file(s) changed", dirty)
    }
    return &FolderMeta{
        Label:   &branch,
        Tagline: &tagline,
        Fields:  []Field{{Key: "branch", Value: branch}},
    }, nil
}
```

**v2: github-pr probe (network)**
```go
func (g *GitHubPRProbe) Read(dir string) (*FolderMeta, error) {
    // Uses `gh pr view --json` — requires gh CLI + auth
    pr := exec("gh", "-C", dir, "pr", "view", "--json", "number,title,state,url")
    // Parse, return as Link + Field
}
```

**v2: jj probe**
```go
func (j *JJProbe) Match(dir string) bool {
    _, err := os.Stat(filepath.Join(dir, ".jj"))
    return err == nil
}
```

**Future: CI probe, test-status probe, custom script probes**

### Script probes (user-configurable)

For probes that don't need to be compiled into gmuxd, users can drop scripts in `~/.config/gmux/probes/`:

```bash
#!/usr/bin/env bash
# ~/.config/gmux/probes/cargo-test.sh
# GMUX_PROBE_MATCH: Cargo.toml
# GMUX_PROBE_INTERVAL: 30

# Script receives directory as $1, outputs JSON to stdout
cd "$1"
result=$(cargo test --no-run 2>&1)
if echo "$result" | grep -q "error"; then
    echo '{"tagline":"build errors","status":"error"}'
else
    echo '{"tagline":"builds clean","status":"success"}'
fi
```

Convention:
- File must be executable
- `GMUX_PROBE_MATCH` comment: filename to check for in the directory (like `Match()`)
- `GMUX_PROBE_INTERVAL` comment: poll interval in seconds (default: 10)
- Receives directory path as `$1`
- Outputs JSON matching `FolderMeta` schema to stdout
- Non-zero exit = probe error (logged, retried next interval)
- Timeout: 10s per invocation (kill if slower)

This makes it trivial to add custom folder intelligence without any Go code. A user who wants "show me if `pnpm test` passes in this directory" writes a 5-line bash script.

### Session-to-folder metadata contribution

Sessions don't need an explicit mechanism to "push" metadata to their folder. Instead:

1. **Status aggregation is automatic.** The folder's status dot reflects the worst status of any child session. A test runner session reporting `status: error` makes the folder dot red. A session reporting `status: attention` makes it pulse.

2. **Session metadata is visible in context.** When you select a session, the header bar shows both the session's metadata AND the folder's probe metadata. You see the full picture: "this agent (session) is working on this branch (git probe) which has this PR (github-pr probe)."

3. **Probes can read session state if needed.** A custom probe could query gmuxd's API to check session statuses within a folder. But this is an advanced pattern, not the default.

This keeps the contract simple: sessions report their own status (via adapters). Folders aggregate and enrich with probes. No bidirectional coupling.

### UI surfaces

Three surfaces render this hierarchy:

**Sidebar — folder headings**
```
▼ gmux                           ● 3
  main · 2 files changed · PR #42
```
- Folder name (directory basename)
- Aggregate status dot + session count
- Tagline from probes (composited: git branch + PR link + custom)
- Collapsible

**Sidebar — session items**
```
  ● fix auth bug                  now
    thinking · pi
```
- Session title
- Status dot + subtitle
- Adapter type badge
- Time indicator

**Header bar (contextual to selected session)**
```
┌─────────────────────────────────────────────────────────────────┐
│ fix auth bug                    ● active · thinking             │
│ ~/dev/gmux · main · pi         [Open PR] [Open in Editor] [⏹] │
└─────────────────────────────────────────────────────────────────┘
```
- Session title + status
- Folder path + branch (from git probe) + adapter type
- Action buttons: links from probes, kill/resume from core, custom from adapters
- This is where "open PR", "open in VS Code", "commit" buttons live

**Mobile responsive:** sidebar is the primary view (full-width), header bar condenses, terminal goes full-screen on session tap.

### Views (future, not in v1)

A view is a named configuration: filter, grouping, sort order, display density. Stored in config, selected via URL param.

```yaml
# ~/.config/gmux/views.yaml
views:
  triage:
    group_by: folder    # default
    sort_by: status     # attention first
    filter: { alive: true }
    collapse: [dead]
  gmux:
    filter: { folder: "*/gmux" }
  all:
    group_by: folder
    collapse: [dead]
default_view: triage
```

Accessible via `/?view=gmux`. Bare URL uses `default_view`. Views are entirely a UI concern — gmuxd serves all sessions regardless, the frontend filters.

v1 ships with one hardcoded view: the triage view. URL params (`?project=gmux`) act as ad-hoc filters. Named views come when we know what knobs people actually want.

### New session creation (future, not in v1)

Adapters and probes can register actions that include "new session" commands:

```go
// In the pi adapter's folder-level contribution:
Action{
    Label:   "New pi session",
    Command: []string{"gmuxr", "--title", "new task", "pi"},
    Icon:    "➕",
}

// In a workspace probe:
Action{
    Label:   "New worktree + session",
    Command: []string{"workspace", "create", "--agent", "pi"},
    Icon:    "🌿",
    Confirm: true,
}
```

These render as buttons in the sidebar folder heading or header bar. Clicking one calls gmuxd, which spawns the command. The new session appears automatically via the normal discovery flow.

This means gmux never hardcodes "how to create a session" — adapters and probes bring that knowledge. The `+` button in a folder shows whatever actions are registered for that folder's probes and adapters.

## Implementation plan

### Phase 1: Folder grouping in UI (v1)
- Sessions grouped by cwd in sidebar (already partially implemented)
- Folder headings: directory basename, aggregate status dot, session count
- Header bar: session title, status, cwd, adapter type, kill button
- No probes yet — folders are just visual grouping

### Phase 2: Probe interface + git probe
- Define `Probe` interface in `services/gmuxd/internal/probe/`
- Implement git probe (branch, dirty state)
- Folder metadata struct, probe registry, polling loop
- Sidebar folder headings show branch + dirty indicator
- Header bar shows branch field

### Phase 3: Script probes
- Script probe loader: scan `~/.config/gmux/probes/`, parse header comments
- Shell out with timeout, parse JSON stdout
- Same `FolderMeta` schema as Go probes

### Phase 4: GitHub PR probe + action buttons
- github-pr probe using `gh` CLI
- Link rendering in sidebar (PR badge) and header bar (open PR button)
- Action button rendering + execution via gmuxd

### Phase 5: Views system
- Config file loader (`~/.config/gmux/views.yaml`)
- URL param `?view=name` routing
- Filter/group/sort/collapse configuration
- View picker UI (if multiple views configured)

## Consequences

### Positive
- **Automatic organization** — folders emerge from session cwds, no manual grouping
- **Incremental enrichment** — git probe adds value immediately, more probes add more
- **Low barrier to extend** — script probes are 5-line bash scripts
- **Clean separation** — adapters manage sessions (active), probes observe directories (passive)
- **Mobile-friendly by default** — triage view works on any screen size
- **No lock-in** — probes are optional enrichment, gmux works without any

### Negative
- Probes shell out to `git`, `gh`, etc. — adds subprocess overhead per folder per interval
- Multiple probes per folder means multiple metadata sources to composite — need clear merge semantics
- Script probes have a trust boundary (user scripts executed by gmuxd)

### Neutral
- Folder-as-cwd means two sessions in different subdirectories of the same repo create two folders (acceptable — the cwd they were launched from is what matters)
- No "project" abstraction — may need one later if worktree grouping proves essential
- Views deferred — URL params are the interim solution
