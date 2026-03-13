import { z } from 'zod'
import { SessionStateSchema, SessionSummarySchema } from './session.js'

const BaseEventSchema = z.object({
  session_id: z.string().min(1),
  updated_at: z.number(),
})

export const SessionUpsertEventSchema = BaseEventSchema.extend({
  type: z.literal('session-upsert'),
  session: SessionSummarySchema,
})

export const SessionStateEventSchema = BaseEventSchema.extend({
  type: z.literal('session-state'),
  state: SessionStateSchema,
})

export const SessionRemoveEventSchema = BaseEventSchema.extend({
  type: z.literal('session-remove'),
})

export const TransportStateEventSchema = BaseEventSchema.extend({
  type: z.literal('transport-state'),
  transport: z.enum(['ttyd']),
  ready: z.boolean(),
})

export const SessionEventSchema = z.discriminatedUnion('type', [
  SessionUpsertEventSchema,
  SessionStateEventSchema,
  SessionRemoveEventSchema,
  TransportStateEventSchema,
])

export type SessionEvent = z.infer<typeof SessionEventSchema>
