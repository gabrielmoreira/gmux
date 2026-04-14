- **Clickable links now work on mobile.** Tapping a URL in terminal output
  previously failed to open it because the touch handler scrolled the viewport
  before the browser could synthesize the mouse events that xterm.js uses for
  link activation. The scroll is now deferred so links resolve at the correct
  position. ([#44](https://github.com/gabrielmoreira/gmux/pull/44))
- **Smarter project grouping.** Sessions are now grouped by shared VCS remote URLs instead of filesystem paths. Two clones of the same repo on different machines (or with different directory names) appear under one project heading. Fork workflows just work: if your fork's `origin` and the upstream repo share any remote URL, they group together. Falls back to workspace root and directory path for repos without remotes. ([#41](https://github.com/gabrielmoreira/gmux/pull/41))
- ### Status indicators redesign

- **Unread indicator is now blue.** The sidebar dot for sessions with unread
  content changed from amber to blue, making it more visible against dark
  backgrounds.
- **Working indicator is now a hollow ring.** Sessions where an agent is
  actively processing show a pulsating ring outline instead of a filled dot,
  reducing visual noise while remaining recognizable.
- **Transient activity indicator for terminals.** Shell sessions that produce
  output briefly show a muted ring that fades after a few seconds, rather than
  permanently marking as unread. Agent sessions (pi, Claude, Codex) only
  trigger unread when the assistant completes a turn.
- **Arrival animation on all unread transitions.** The grow-pulse animation
  now fires whenever a session becomes unread (previously only on
  working-to-unread transitions). The mobile hamburger badge re-animates
  when additional sessions become unread. ([#46](https://github.com/gabrielmoreira/gmux/pull/46))
- ### Security

- **Fixed unauthenticated localhost listener.** The TCP listener on `localhost:8790` previously required no authentication, which was exploitable via VS Code port forwarding, `docker -p`, and SSH tunnels. All TCP connections now require a bearer token. ([#40](https://github.com/gmuxapp/gmux/pull/40))

### Architecture

- **Unix socket for local IPC.** The `gmux` CLI and `gmuxd` now communicate via a Unix socket (`~/.local/state/gmux/gmuxd.sock`) instead of an unauthenticated TCP connection. Unix sockets cannot be forwarded by VS Code, Docker, or SSH. File permissions (0600/0700) enforce locality.
- **Single authenticated TCP listener.** The TCP listener (default `127.0.0.1:8790`) serves the web UI and API with bearer token authentication on every request. The bind address is controlled by the `GMUXD_LISTEN` env var for container use.

### CLI changes

- `gmuxd` (no args) or `gmuxd start` starts the daemon, always replacing any existing instance.
- `gmuxd stop` replaces `gmuxd shutdown`.
- `gmuxd status` shows daemon health, listen address, and socket path.
- `gmuxd auth` replaces `gmuxd auth-link`, prints the token and a ready-to-use URL.

### Config simplification

- Removed the `[network]` config section and `listen` field. Bind address is now `GMUXD_LISTEN` env var only.
- Removed `GMUXD_PORT`, `GMUXD_ADDR`, and `GMUXD_SOCKET` environment variables. ([#43](https://github.com/gabrielmoreira/gmux/pull/43))
