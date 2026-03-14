# ADR-0009: Session file attribution via content similarity

- Status: Proposed
- Date: 2026-03-15

## Context

When a tool like pi writes session files to disk, gmux needs to know which
running session produced which file. This is the "attribution problem" —
mapping a live gmuxr process to its session file so we can:

1. Set the session's `resume_key` (for deduplication with resumable entries)
2. Optionally feed the matched file back to the runner's sidecar for
   richer state monitoring (watching the file for status changes)

### Why this is hard

- Pi doesn't hold the file open (open → append → close per write)
- Pi doesn't create the file until the first assistant response
- Pi can switch session files mid-process (`/resume`, `/fork`, `/new`)
- Multiple pi instances can share the same cwd and session directory
- We don't want to modify pi's command line or depend on pi cooperation
- `inotify` doesn't report writer PID; `fanotify` does but is Linux-only

### Previous approaches considered

- **`/proc/<pid>/fd` scanning**: unreliable — pi doesn't hold file open
- **Timing correlation**: works for single instance but ambiguous for
  multi-instance (both have recent PTY activity when file appears)
- **fanotify PID reporting**: Linux-only, needs `CAP_SYS_ADMIN` pre-5.13

## Decision

### Content similarity matching

When a session file is written, compare its content against each live
session's terminal scrollback. The session with the most similar recent
output is attributed to that file.

This approach is:
- **Format-agnostic**: doesn't depend on specific JSONL schema versions
- **Self-correcting**: if `/resume` switches files, the next write
  re-attributes correctly without detecting the command
- **Efficient**: compare tails only, runs only on file write events
- **General**: the matching function is reusable across adapters

### Architecture

```
gmuxd (per-machine):
  SessionDirWatcher (one per unique cwd with live sessions):
    - inotify on session directory
    - on IN_CLOSE_WRITE:
        1. single live session in this cwd? → trivially attribute
        2. multiple? → content similarity match
    - sets resume_key on the attributed session
    - notifies the runner of its matched file (reverse channel)

gmuxr (per-session):
  Pi adapter Monitor():
    - spinner detection → active/idle status
    - no file watching, no attribution
  Pi adapter file sidecar (optional, activated after attribution):
    - monitors the MATCHED file for richer state changes
    - only starts once gmuxd tells it which file is "theirs"
```

### Two-layer design

**Layer 1: General tail-similarity matcher (reusable library)**

```go
// SimilarityMatch finds which candidate best matches the probe text.
// Returns the index of the best match, or -1 if no match exceeds
// the minimum threshold.
func SimilarityMatch(probe string, candidates []string, minScore float64) int
```

Both inputs are pre-normalized text. The matcher doesn't know about
JSONL or terminal escapes — it just compares strings. This makes it
reusable for any adapter that needs to correlate file content with
terminal output.

The matching strategy works from the tail: the most recent content in
the file corresponds to the most recent content on screen. A suffix
overlap search is more efficient and more discriminating than full-text
comparison.

**Layer 2: Adapter-specific text extraction (per-adapter)**

Each adapter provides a function to extract matchable text from its
file format:

```go
// ExtractMatchText reads a session file and returns normalized text
// suitable for similarity matching against terminal scrollback.
type TextExtractor func(filePath string) (string, error)
```

For pi, this reads the JSONL file and extracts text content from
message entries — user messages, assistant responses, tool output.
JSON syntax (braces, quotes, keys) is stripped. What remains is the
conversation text that also appears in the terminal.

For the scrollback side, terminal output is normalized by stripping
ANSI escape sequences and collapsing whitespace. The scrollback will
contain TUI chrome (status line, borders) that won't appear in the
file, but the conversation content is present in both and that's what
matches.

### Attribution lifecycle

1. **No file yet**: session has no resume_key. Resumable entry (if any)
   is not deduplicated.

2. **File created**: gmuxd sees `IN_CREATE` + `IN_CLOSE_WRITE`. If one
   session in the cwd → trivially attributed. If multiple → wait for
   content and run similarity match.

3. **Attributed**: resume_key set. gmuxd notifies the runner which file
   is theirs. Runner can optionally start monitoring that specific file.

4. **File switches** (`/resume`, `/fork`, `/new`): the old file stops
   getting writes. A different file starts getting writes. The next
   `IN_CLOSE_WRITE` triggers a new similarity match. Attribution
   updates naturally. No need to detect the switch command.

5. **Session exits**: attribution is moot. The resume_key persists in
   the dead session entry for deduplication with the resumable scan.

### Sticky attribution

Once a session is attributed to a file, re-matching is only needed
when a DIFFERENT file in the same directory gets written. If the same
file is written again, the attribution holds. This means the similarity
match runs rarely after initial attribution.

### Reverse channel: gmuxd → gmuxr

gmuxd needs to tell gmuxr which file was attributed to it. This breaks
the one-way data flow (runner → gmuxd) but is justified:

- The runner's sidecar can do richer monitoring with the specific file
  (e.g., watch for status changes, extract conversation metadata)
- The runner already exposes HTTP endpoints on its socket; gmuxd can
  `POST /file-match` or similar

This is an optional enhancement. The MVP attribution works entirely
in gmuxd. The reverse notification enables the sidecar to do more, but
isn't required for resume_key to function.

### gmuxr sidecar role (post-attribution)

Once the runner knows its session file, the pi adapter sidecar can:
- Watch that single file for changes (one inotify watch, not the whole dir)
- Parse new JSONL entries for richer status (thinking, tool use, errors)
- Update session title/subtitle from conversation content

This monitoring is precise (one file, known to be ours) and cheap.
It's the per-session enrichment layer on top of gmuxd's global
attribution layer.

## Matching details

### Scrollback normalization

```
Raw scrollback (bytes from PTY):
  \x1b[1m\x1b[34mHere's how to fix that:\x1b[0m\n  ...

Normalized:
  Here's how to fix that: ...
```

Strip: ANSI CSI sequences, OSC sequences, control characters.
Collapse: runs of whitespace to single space. Trim.

### File content normalization (pi JSONL)

```
Raw JSONL line:
  {"type":"message","id":"abc","message":{"role":"assistant","content":[{"type":"text","text":"Here's how to fix that:"}]}}

Extracted text:
  Here's how to fix that:
```

The pi extractor reads JSONL lines, finds `"text"` values in content
arrays, concatenates them. This is the only pi-specific part — the
JSONL structure with `type`, `message`, `content[].text` is stable
across pi versions.

### Matching algorithm

Compare the tail of the extracted file text against the tail of each
candidate session's scrollback:

1. Take last ~500 chars of file text (normalized)
2. For each live session in the cwd:
   a. Get last ~2000 chars of scrollback (normalized)
   b. Find longest common substring between file tail and scrollback tail
   c. Score = length of longest match / length of file tail
3. Best score above threshold → attributed

The asymmetric window (500 vs 2000) accounts for the scrollback having
more content (TUI chrome, older messages) than the file tail.

## Consequences

### Positive
- **No pi changes required**: works with any pi version
- **Self-correcting**: `/resume`, `/fork` handled automatically
- **General**: matcher function reusable for other adapters
- **Efficient**: runs only on file writes, compares tails only
- **Resilient**: doesn't depend on specific output formats

### Negative
- **Latency**: attribution happens on first file write (after first
  assistant response), not at session start
- **Reverse channel**: gmuxd → gmuxr notification adds bidirectional
  communication (manageable, optional for MVP)
- **Scrollback access**: gmuxd needs to read session scrollback,
  either via cached WS relay or a new runner endpoint

### Neutral
- Single instance per cwd (common case) doesn't need similarity matching
- Multi-instance edge case handled correctly but with one write delay
- Pi-specific JSONL extraction is minimal (~20 lines of code)
