import { describe, expect, it } from 'vitest'
import {
  SessionEventSchema,
  SessionSchema,
  StatusStateSchema,
  successEnvelope,
} from './index.js'

describe('protocol schemas', () => {
  it('parses session (schema v2)', () => {
    const result = SessionSchema.parse({
      id: 'sess-1',
      kind: 'pi',
      alive: true,
      pid: 12345,
      title: 'test session',
      status: { label: 'thinking', state: 'active' },
      resize_owner_id: 'device-1',
      terminal_cols: 120,
      terminal_rows: 40,
    })

    expect(result.id).toBe('sess-1')
    expect(result.alive).toBe(true)
    expect(result.status?.state).toBe('active')
    expect(result.resize_owner_id).toBe('device-1')
    expect(result.terminal_cols).toBe(120)
    expect(result.terminal_rows).toBe(40)
  })

  it('parses session with null status', () => {
    const result = SessionSchema.parse({
      id: 'sess-2',
      kind: 'generic',
      alive: false,
      status: null,
    })

    expect(result.status).toBeNull()
    expect(result.alive).toBe(false)
  })

  it('validates session-upsert event', () => {
    const event = SessionEventSchema.parse({
      type: 'session-upsert',
      id: 'sess-1',
      session: {
        id: 'sess-1',
        kind: 'pi',
        alive: true,
        status: { label: 'running', state: 'active' },
      },
    })

    expect(event.type).toBe('session-upsert')
    if (event.type === 'session-upsert') {
      expect(event.session.alive).toBe(true)
    }
  })

  it('validates session-remove event', () => {
    const event = SessionEventSchema.parse({
      type: 'session-remove',
      id: 'sess-1',
    })
    expect(event.type).toBe('session-remove')
  })

  it('validates status states', () => {
    for (const state of ['active', 'attention', 'success', 'error', 'paused', 'info']) {
      expect(StatusStateSchema.parse(state)).toBe(state)
    }
  })

  it('builds typed success envelopes', () => {
    const Schema = successEnvelope(StatusStateSchema)
    const parsed = Schema.parse({ ok: true, data: 'active' })
    expect(parsed.data).toBe('active')
  })
})
