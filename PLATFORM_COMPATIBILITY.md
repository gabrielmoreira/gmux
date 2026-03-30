# Platform Compatibility

This document captures the main implementation differences between Windows and Unix in `gmux`, along with the current support level and known limitations.
## Goal

Track the main implementation differences between Windows and Unix so the Windows port stays honest about what is truly cross-platform and what is platform-specific.

## High-level status

- **Windows:** initial native support is working.
- **Unix:** behavior remains supported and was rechecked after the Windows changes.
- **Primary validated Windows path:** Windows Terminal + PowerShell 7 / Windows PowerShell.
- **Best-effort Windows paths:** Git Bash and WSL launchers.

## Main implementation differences

| Area | Windows | Unix | Why it differs | Current note |
| --- | --- | --- | --- | --- |
| PTY backend | `github.com/aymanbagabas/go-pty` via ConPTY | `github.com/creack/pty` | Windows has no native POSIX PTY model | Shared `ptyserver` uses OS-specific files |
| Child detach / re-exec | `DETACHED_PROCESS` + `CREATE_NEW_PROCESS_GROUP` | `Setsid: true` | Windows has no `setsid(2)` | Split into `exec_windows.go` / `exec_unix.go` |
| Terminal resize | ConPTY resize directly; no `SIGWINCH` | PTY resize + `SIGWINCH` forwarding | Windows has no POSIX signal resize flow | Windows and Unix resize paths are intentionally different |
| Process teardown | Kill the process directly | Kill the process group when appropriate | Process-group semantics differ across platforms | Unix setup-failure cleanup uses hard group kill |
| Local terminal sizing | Prefer `stdout`, fallback `stdin` | Prefer `stdin`, fallback `stdout` | Detached Windows consoles can make stdin unreliable; Unix controlling TTY is typically stdin | Kept platform-specific to avoid Unix regressions |
| Daemon state dir | `%LOCALAPPDATA%\gmux` | `$XDG_STATE_HOME/gmux` or `~/.local/state/gmux` | Windows runtime-data conventions differ | Documented in config docs |
| Config / theme / keybind paths | `%USERPROFILE%\.config\gmux\...` | `$XDG_CONFIG_HOME/gmux/...` or `~/.config/gmux/...` | Current repo still uses XDG-style config semantics on Unix and a matching Windows fallback path | User-facing docs updated |
| Session sockets | `%TEMP%\gmux-sessions` | `/tmp/gmux-sessions` | Temp directory conventions differ | Still uses AF_UNIX on both |
| Browser launch fallback | `rundll32 url.dll,FileProtocolHandler` | normal Unix browser opener path | Windows browser opening differs | Windows fallback added |
| PI / OMP session discovery | Windows-encoded path names under `%USERPROFILE%\.pi` / `%USERPROFILE%\.omp` | Unix-encoded path names under `~/.pi` / `~/.omp` | Session directory naming differs by tool and path shape | Encoding logic is tested on both shapes |

## Windows

### What is supported now

- `gmuxd start --replace` runs natively on Windows.
- `gmux` can launch and attach interactive sessions without WSL.
- Browser UI opens and can attach to live sessions.
- PI / OMP resumable sessions are discovered and resumed on Windows.
- Windows-specific launchers are exposed for:
  - Shell
  - PowerShell 7
  - PowerShell 5
  - cmd.exe
  - WSL
  - Git Bash
  - Codex
  - GitHub Copilot
  - Gemini

### Known limitations

- Windows support is still **initial**, not feature-identical in every low-level terminal detail.
- Git Bash and WSL are exposed as launchers, but their behavior depends on the launched shell/tool, not just gmux.
- Third-party agent CLIs may still have their own Windows limitations.
- `SIGWINCH`-style behavior is inherently Unix-specific, so Windows resize coverage is validated through ConPTY behavior rather than signal delivery semantics.

## Unix

### What was explicitly protected during the port

- PTY command argument handling still strips `argv[0]` correctly before process launch.
- Terminal sizing still follows the controlling TTY (`stdin`) first.
- Listener setup failure still tears down the started process safely.
- State/config/temp path behavior remains Unix-appropriate.

### Unix-specific behavior that remains intentional

- Session/process-group semantics still rely on POSIX signals.
- `SIGWINCH` forwarding remains the Unix resize model.
- Unix test coverage still exercises the signal-based resize path directly.

## Verification snapshot

Validated during this pass:

- `go test ./cmd/gmux ./internal/ptyserver` in `cli/gmux`
- `go test ./adapters -run "TestShellLaunchers$|TestSessionRootDir|TestOmpSessionRootDir|TestPiMatchDirect$|TestSessionDirEncoding$"` in `packages/adapter`
- `go test ./internal/discovery ./internal/sessionfiles ./cmd/gmuxd -run "TestNotifyNewSessionUsesWindowsOmpSessionDir$|TestScanHandlesShortSessionIDs$|TestScanSkipsEmptySessionIDs$|TestDiscoverLaunchersUsesCompiledAdapters$"` in `services/gmuxd`
- `GOOS=linux GOARCH=amd64 go build ./cmd/gmux`
- `GOOS=linux GOARCH=amd64 go build ./cmd/gmuxd`
- `pnpm build` in `apps/website`

## Scope

This document is meant as an engineering-facing compatibility reference. User-facing installation and troubleshooting details remain in:

- `README.md`
- `apps/website/src/content/docs/quick-start.mdx`
- `apps/website/src/content/docs/configuration.md`
- `apps/website/src/content/docs/troubleshooting.md`
