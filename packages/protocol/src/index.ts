import { z } from 'zod'

export const SessionStateSchema = z.enum([
  'starting',
  'running',
  'waiting',
  'idle',
  'exited',
  'error',
])

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
export type SessionMetadata = z.infer<typeof SessionMetadataSchema>
