import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { usePublishStore } from '../publish'

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

const store = () => usePublishStore.getState()

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

describe('publish store', () => {
  it('streams lines and succeeds', async () => {
    stubFetch(202, { started: true })
    await store().start()
    expect(store().status).toBe('running')

    const es = FakeSource.last!
    expect(es.url).toContain('/api/publish/stream')
    es.emit({ kind: 'line', line: 'Copying' })
    es.emit({ kind: 'done', ok: true })

    expect(store().lines).toEqual(['Copying'])
    expect(store().status).toBe('succeeded')
    expect(es.closed).toBe(true)
  })

  it('shows the error from a failed publish', async () => {
    stubFetch(202, { started: true })
    await store().start()
    FakeSource.last!.emit({ kind: 'done', ok: false, error: 'git push failed' })

    expect(store().status).toBe('failed')
    expect(store().error).toBe('git push failed')
  })

  it('fails when the request is refused', async () => {
    stubFetch(422, { code: 'invalid_publish_settings', message: 'Check the publish settings: x.' })
    await store().start()

    expect(store().status).toBe('failed')
    expect(store().error).toBe('Check the publish settings: x.')
    expect(FakeSource.last).toBeNull()
  })
})
