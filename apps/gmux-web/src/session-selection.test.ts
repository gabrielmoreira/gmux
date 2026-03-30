import { describe, expect, it } from 'vitest'
import { findFirstAttachableSession } from './session-selection'
import type { Session } from './types'

function makeSession(overrides: Partial<Session> & { id: string }): Session {
  return {
    id: overrides.id,
    created_at: overrides.created_at ?? '2026-03-30T00:00:00Z',
    command: overrides.command ?? ['cmd.exe'],
    cwd: overrides.cwd ?? 'C:/work',
    workspace_root: overrides.workspace_root,
    kind: overrides.kind ?? 'shell',
    alive: overrides.alive ?? false,
    pid: overrides.pid ?? null,
    exit_code: overrides.exit_code ?? null,
    started_at: overrides.started_at ?? '',
    exited_at: overrides.exited_at ?? null,
    title: overrides.title ?? overrides.id,
    subtitle: overrides.subtitle ?? '',
    status: overrides.status ?? null,
    unread: overrides.unread ?? false,
    resumable: overrides.resumable,
    resume_key: overrides.resume_key,
    socket_path: overrides.socket_path ?? '',
    terminal_cols: overrides.terminal_cols,
    terminal_rows: overrides.terminal_rows,
    shell_title: overrides.shell_title,
    adapter_title: overrides.adapter_title,
    binary_hash: overrides.binary_hash,
    stale: overrides.stale,
  }
}

describe('findFirstAttachableSession', () => {
  it('returns the first alive session with a socket path', () => {
    const sessions = [
      makeSession({ id: 'dead', alive: false, socket_path: 'dead.sock' }),
      makeSession({ id: 'no-socket', alive: true, socket_path: '' }),
      makeSession({ id: 'attachable', alive: true, socket_path: 'live.sock' }),
      makeSession({ id: 'later', alive: true, socket_path: 'later.sock' }),
    ]

    expect(findFirstAttachableSession(sessions)?.id).toBe('attachable')
  })

  it('returns null when no session is attachable yet', () => {
    const sessions = [
      makeSession({ id: 'dead', alive: false, socket_path: 'dead.sock' }),
      makeSession({ id: 'waiting', alive: true, socket_path: '' }),
    ]

    expect(findFirstAttachableSession(sessions)).toBeNull()
  })
})
