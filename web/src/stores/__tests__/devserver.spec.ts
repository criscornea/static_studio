import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useDevServerStore } from '../devserver'

const store = () => useDevServerStore.getState()

function jsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

beforeEach(() => {
  vi.useFakeTimers()
  store().reset()
})

afterEach(() => {
  store().reset()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('dev server store', () => {
  it('starts and polls until ready', async () => {
    const url = 'http://127.0.0.1:4000'
    const fetchMock = vi
      .fn<() => Promise<Response>>()
      .mockResolvedValueOnce(jsonResponse(202, { running: true, ready: false, url }))
      .mockResolvedValueOnce(jsonResponse(200, { running: true, ready: true, url }))
    vi.stubGlobal('fetch', fetchMock)

    await store().start()
    expect(store().running).toBe(true)
    expect(store().ready).toBe(false)

    await vi.advanceTimersByTimeAsync(600)
    expect(store().ready).toBe(true)
    expect(store().url).toBe(url)
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('reports start errors', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(jsonResponse(500, { code: 'devserver_failed', message: 'no hugo' })),
      ),
    )
    await store().start()
    expect(store().running).toBe(false)
    expect(store().error).not.toBeNull()
  })

  it('stop clears state', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(new Response(null, { status: 204 }))))
    useDevServerStore.setState({ running: true, ready: true, url: 'http://127.0.0.1:1' })
    await store().stop()
    expect(store().running).toBe(false)
    expect(store().url).toBeNull()
  })
})
