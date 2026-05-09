import type { Session } from './types'

export function findFirstAttachableSession(sessions: Session[]): Session | null {
  return sessions.find((session) => session.alive && !!session.socket_path) ?? null
}
