import { serve } from '@hono/node-server'
import { Hono } from 'hono'

const app = new Hono()

app.get('/health', (c) => c.json({ ok: true, service: 'gmux-api' }))

const port = Number(process.env.PORT ?? 8787)

serve({ fetch: app.fetch, port }, () => {
  console.log(`gmux-api listening on :${port}`)
})
