import { z } from 'zod'

export const SessionStateSchema = z.enum([
  'starting',
  'running',
  'waiting',
  'idle',
  'exited',
  'error',
])

export const SessionSummarySchema = z.object({
  session_id: z.string().min(1),
  abduco_name: z.string().min(1),
  title: z.string().optional(),
  kind: z.enum(['pi', 'generic', 'opencode']).default('pi'),
  state: SessionStateSchema,
  updated_at: z.number(),
})

export const AttachResponseSchema = z.object({
  transport: z.enum(['ttyd']),
  port: z.number().int().positive(),
  is_new: z.boolean(),
  token: z.string().optional(),
})

export const SessionMetadataSchema = z.object({
  version: z.literal(1),
  session_id: z.string().min(1),
  abduco_name: z.string().min(1),
  kind: z.enum(['pi', 'generic', 'opencode']),
  command: z.array(z.string()).min(1),
  cwd: z.string().min(1),
  state: SessionStateSchema,
  created_at: z.number(),
  updated_at: z.number(),
})

export type SessionState = z.infer<typeof SessionStateSchema>
export type SessionSummary = z.infer<typeof SessionSummarySchema>
export type AttachResponse = z.infer<typeof AttachResponseSchema>
export type SessionMetadata = z.infer<typeof SessionMetadataSchema>
