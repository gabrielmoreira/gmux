import { createTRPCProxyClient, httpBatchLink } from '@trpc/client'
import type { SessionEvent, SessionSummary } from '@gmux/protocol'
import { SessionEventSchema } from '@gmux/protocol'
import { render } from 'preact'
import { useCallback, useEffect, useMemo, useRef, useState } from 'preact/hooks'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { AttachAddon } from '@xterm/addon-attach'
import '@xterm/xterm/css/xterm.css'

const trpc: any = createTRPCProxyClient<any>({
  links: [httpBatchLink({ url: '/trpc' })],
})

function sortSessions(items: SessionSummary[]) {
  return [...items].sort((a, b) => b.updated_at - a.updated_at)
}

const STATE_COLORS: Record<string, string> = {
  running: '#a3be8c',
  waiting: '#ebcb8b',
  starting: '#81a1c1',
  idle: '#88c0d0',
  exited: '#4c566a',
  error: '#bf616a',
}

const NORD_THEME = {
  background: '#2e3440',
  foreground: '#d8dee9',
  cursor: '#d8dee9',
  cursorAccent: '#2e3440',
  selectionBackground: '#434c5ecc',
  black: '#3b4252',
  red: '#bf616a',
  green: '#a3be8c',
  yellow: '#ebcb8b',
  blue: '#81a1c1',
  magenta: '#b48ead',
  cyan: '#88c0d0',
  white: '#e5e9f0',
  brightBlack: '#4c566a',
  brightRed: '#bf616a',
  brightGreen: '#a3be8c',
  brightYellow: '#ebcb8b',
  brightBlue: '#81a1c1',
  brightMagenta: '#b48ead',
  brightCyan: '#8fbcbb',
  brightWhite: '#eceff4',
}

function StateDot({ state }: { state: string }) {
  return (
    <span
      style={{
        display: 'inline-block',
        width: 8,
        height: 8,
        borderRadius: '50%',
        background: STATE_COLORS[state] ?? '#4c566a',
        marginRight: 6,
      }}
    />
  )
}

function TerminalView({ sessionId }: { sessionId: string }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    if (!containerRef.current) return

    const term = new Terminal({
      theme: NORD_THEME,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      fontSize: 14,
      cursorBlink: true,
    })
    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)

    term.open(containerRef.current)
    fitAddon.fit()

    // Connect WebSocket through gmuxd proxy
    const wsProtocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${wsProtocol}//${location.host}/ws/${sessionId}`)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = () => {
      const attachAddon = new AttachAddon(ws)
      term.loadAddon(attachAddon)

      // Send initial resize
      const dims = fitAddon.proposeDimensions()
      if (dims) {
        ws.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }))
      }
    }

    ws.onclose = () => {
      term.write('\r\n\x1b[90m[session disconnected]\x1b[0m\r\n')
    }

    termRef.current = term

    // Handle window resize
    const onResize = () => {
      fitAddon.fit()
      const dims = fitAddon.proposeDimensions()
      if (dims && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }))
      }
    }
    window.addEventListener('resize', onResize)

    return () => {
      window.removeEventListener('resize', onResize)
      ws.close()
      term.dispose()
      termRef.current = null
      wsRef.current = null
    }
  }, [sessionId])

  return (
    <div
      ref={containerRef}
      style={{
        flex: 1,
        minHeight: 300,
        background: '#2e3440',
        borderRadius: '4px',
        overflow: 'hidden',
      }}
    />
  )
}

function App() {
  const [sessions, setSessions] = useState<SessionSummary[]>([])
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true
    trpc.sessions.list
      .query()
      .then((data: SessionSummary[]) => {
        if (!active) return
        setSessions(sortSessions(data))
        if (!selectedSessionId && data.length > 0) {
          setSelectedSessionId(data[0].session_id)
        }
      })
      .catch((err: unknown) => {
        if (!active) return
        setError(String(err))
      })
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    const source = new EventSource('/api/events')

    source.onmessage = (message) => {
      let parsed: SessionEvent
      try {
        parsed = SessionEventSchema.parse(JSON.parse(message.data))
      } catch {
        return
      }

      if (parsed.type === 'session-upsert') {
        setSessions((current) => {
          const without = current.filter((it) => it.session_id !== parsed.session_id)
          return sortSessions([...without, parsed.session])
        })
        return
      }

      if (parsed.type === 'session-state') {
        setSessions((current) =>
          sortSessions(
            current.map((it) =>
              it.session_id === parsed.session_id
                ? { ...it, state: parsed.state, updated_at: parsed.updated_at }
                : it,
            ),
          ),
        )
        return
      }

      if (parsed.type === 'session-remove') {
        setSessions((current) => current.filter((it) => it.session_id !== parsed.session_id))
      }
    }

    source.onerror = () => setError('event stream disconnected')
    return () => source.close()
  }, [])

  const selected = useMemo(
    () => sessions.find((it) => it.session_id === selectedSessionId) ?? null,
    [sessions, selectedSessionId],
  )

  const canAttach = selected && selected.state !== 'exited' && selected.state !== 'error'

  return (
    <div
      style={{
        fontFamily: 'system-ui, sans-serif',
        background: '#2e3440',
        color: '#eceff4',
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <header style={{ padding: '0.5rem 1rem', borderBottom: '1px solid #3b4252' }}>
        <h1 style={{ color: '#88c0d0', margin: 0, fontSize: '1.2rem' }}>gmux</h1>
        {error ? (
          <span style={{ color: '#bf616a', fontSize: '0.8rem', marginLeft: '1rem' }}>⚠ {error}</span>
        ) : null}
      </header>

      <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        <aside
          style={{
            width: 240,
            borderRight: '1px solid #3b4252',
            padding: '0.5rem',
            overflowY: 'auto',
          }}
        >
          {sessions.length === 0 ? <p style={{ opacity: 0.6, padding: '0.5rem' }}>No sessions</p> : null}
          {sessions.map((session) => {
            const isSelected = selectedSessionId === session.session_id
            return (
              <button
                key={session.session_id}
                onClick={() => setSelectedSessionId(session.session_id)}
                style={{
                  width: '100%',
                  textAlign: 'left',
                  marginBottom: '0.25rem',
                  cursor: 'pointer',
                  border: isSelected ? '1px solid #88c0d0' : '1px solid transparent',
                  borderRadius: '6px',
                  background: isSelected ? '#3b4252' : 'transparent',
                  color: '#eceff4',
                  padding: '0.5rem 0.625rem',
                }}
              >
                <div style={{ fontWeight: 600, fontSize: '0.85rem' }}>
                  {session.title ?? session.abduco_name}
                </div>
                <div style={{ fontSize: '0.75rem', opacity: 0.7, marginTop: 2 }}>
                  <StateDot state={session.state} />
                  {session.state}
                </div>
              </button>
            )
          })}
        </aside>

        <main style={{ flex: 1, display: 'flex', flexDirection: 'column', padding: '0.5rem' }}>
          {selected ? (
            <>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.5rem',
                  marginBottom: '0.5rem',
                  fontSize: '0.85rem',
                }}
              >
                <StateDot state={selected.state} />
                <strong>{selected.title ?? selected.abduco_name}</strong>
                <span style={{ opacity: 0.5 }}>{selected.session_id}</span>
                <span style={{ opacity: 0.5 }}>{selected.kind}</span>
              </div>
              {canAttach ? (
                <TerminalView key={selected.session_id} sessionId={selected.session_id} />
              ) : (
                <div
                  style={{
                    flex: 1,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    opacity: 0.5,
                  }}
                >
                  Session {selected.state}
                </div>
              )}
            </>
          ) : (
            <div
              style={{
                flex: 1,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                opacity: 0.5,
              }}
            >
              Select a session
            </div>
          )}
        </main>
      </div>
    </div>
  )
}

render(<App />, document.getElementById('app')!)
