---
bump: minor
---

### Windows

- **Initial native Windows support.** `gmux` and `gmuxd` now run without WSL in the primary Windows flow, with Windows-aware binary discovery, daemon/runtime paths, console cleanup, launcher options, and updated docs for current limitations.
- **Safer cross-platform behavior.** The Windows port no longer regresses Unix PTY argument handling or terminal sizing, and the portability tests now avoid machine-specific filesystem assumptions.
