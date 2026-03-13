import { z } from 'zod'

export const ErrorCodeSchema = z.enum([
  'bad_request',
  'unauthorized',
  'forbidden',
  'not_found',
  'conflict',
  'unavailable',
  'internal_error',
])

export const ErrorBodySchema = z.object({
  code: ErrorCodeSchema,
  message: z.string(),
})

export function successEnvelope<T extends z.ZodTypeAny>(schema: T) {
  return z.object({
    ok: z.literal(true),
    data: schema,
  })
}

export const ErrorEnvelopeSchema = z.object({
  ok: z.literal(false),
  error: ErrorBodySchema,
})

export function responseEnvelope<T extends z.ZodTypeAny>(schema: T) {
  return z.union([successEnvelope(schema), ErrorEnvelopeSchema])
}

export type ErrorCode = z.infer<typeof ErrorCodeSchema>
export type ErrorBody = z.infer<typeof ErrorBodySchema>
