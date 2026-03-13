import { describe, expect, it } from 'vitest'
import {
  SessionEventSchema,
  SessionStateSchema,
  SessionSummarySchema,
  successEnvelope,
} from './index.js'

describe('protocol schemas', () => {
  it('parses session summary', () => {
    const result = SessionSummarySchema.parse({
      session_id: 'sess-1',
      abduco_name: 'pi:demo:1',
      kind: 'pi',
      state: 'running',
      updated_at: Date.now() / 1000,
    })

    expect(result.session_id).toBe('sess-1')
  })

  it('validates event union by type', () => {
    const event = SessionEventSchema.parse({
      type: 'session-state',
      session_id: 'sess-1',
      state: 'waiting',
      updated_at: Date.now() / 1000,
    })

    expect(event.type).toBe('session-state')
    expect(event.state).toBe('waiting')
  })

  it('builds typed success envelopes', () => {
    const Schema = successEnvelope(SessionStateSchema)
    const parsed = Schema.parse({ ok: true, data: 'idle' })
    expect(parsed.data).toBe('idle')
  })
})
