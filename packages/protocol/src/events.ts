import { z } from 'zod'
import { SessionSchema } from './session.js'

const BaseEventSchema = z.object({
  id: z.string().min(1),
})

export const SessionUpsertEventSchema = BaseEventSchema.extend({
  type: z.literal('session-upsert'),
  session: SessionSchema,
})

export const SessionRemoveEventSchema = BaseEventSchema.extend({
  type: z.literal('session-remove'),
})

export const SessionEventSchema = z.discriminatedUnion('type', [
  SessionUpsertEventSchema,
  SessionRemoveEventSchema,
])

export type SessionEvent = z.infer<typeof SessionEventSchema>
