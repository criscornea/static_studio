import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useBuildStore } from '../build'

class FakeSource {
  static last: FakeSource | null = null
  onmessage: ((m: MessageEvent<string>) => void) | null = null
  onerror: (() => void) | null = null
  closed = false
  constructor(public url: string) {
    FakeSource.last = this
  }
  close() {
    this.closed = true
  }
  emit(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent<string>)
  }
}

const store = () => useBuildStore.getState()

beforeEach(() => {
  vi.stubGlobal('EventSource', FakeSource)
  FakeSource.last = null
  store().reset()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

function stubFetch(status: number, body: unknown) {
  vi.stubGlobal(
    'fetch',
    vi.fn(() =>
      Promise.resolve(
        new Response(JSON.stringify(body), {
          status,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    ),
  )
}

describe('build store', () => {
  it('collects lines and finishes on done', async () => {
    stubFetch(202, { started: true })
    await store().start()
    expect(store().status).toBe('running')

    const es = FakeSource.last!
    expect(es.url).toContain('/api/build/stream')
    es.emit({ kind: 'line', line: 'one' })
    es.emit({ kind: 'line', line: 'two' })
    es.emit({ kind: 'done', ok: true })

    expect(store().lines).toEqual(['one', 'two'])
    expect(store().status).toBe('succeeded')
    expect(es.closed).toBe(true)
  })

  it('reports a failed build', async () => {
    stubFetch(202, { started: true })
    await store().start()
    FakeSource.last!.emit({ kind: 'done', ok: false, error: 'exit status 1' })

    expect(store().status).toBe('failed')
    expect(store().error).toBe('exit status 1')
  })

  it('reports a refused start', async () => {
    stubFetch(409, { code: 'build_running', message: 'A build is already running.' })
    await store().start()

    expect(store().status).toBe('failed')
    expect(store().error).toBe('A build is already running.')
    expect(FakeSource.last).toBeNull()
  })
})
