import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { SearchResults } from '@/api/types'
import { searchDelay, useSearchStore } from '../search'
import { useProjectStore } from '../project'

const store = () => useSearchStore.getState()
const searchURL = (q: string) => `/api/search?q=${encodeURIComponent(q)}`

const results = (...paths: string[]): SearchResults => ({
  hits: paths.map((path) => ({ path, title: path, draft: false })),
  truncated: false,
})

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

/** A request the fake fetch is holding until the test responds to it. */
interface Pending {
  url: string
  resolve: (res: Response) => void
}

let pending: Pending[] = []

function respond(url: string, res: Response) {
  const i = pending.findIndex((p) => p.url === url)
  const [req] = i < 0 ? [] : pending.splice(i, 1)
  if (req === undefined) {
    throw new Error(`no pending request for ${url}`)
  }
  req.resolve(res)
}

/** Lets the debounce fire, then waits for the response handling to finish. */
async function pause() {
  await vi.advanceTimersByTimeAsync(searchDelay)
}

beforeEach(() => {
  vi.useFakeTimers()
  pending = []
  vi.stubGlobal(
    'fetch',
    (input: RequestInfo) =>
      new Promise<Response>((resolve) => {
        pending.push({ url: String(input), resolve })
      }),
  )
  store().clear()
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('setQuery', () => {
  it('shows the text and a busy state at once, and asks the backend after the pause', async () => {
    store().setQuery('golang')

    expect(store().query).toBe('golang')
    expect(store().loading).toBe(true)
    expect(pending).toHaveLength(0)

    await pause()
    expect(pending.map((p) => p.url)).toEqual([searchURL('golang')])

    respond(searchURL('golang'), json(200, results('content/a.md')))
    await vi.advanceTimersByTimeAsync(0)

    expect(store().results?.hits).toHaveLength(1)
    expect(store().loading).toBe(false)
  })

  it('sends one request for a burst of keystrokes', async () => {
    store().setQuery('g')
    await vi.advanceTimersByTimeAsync(100)
    store().setQuery('go')
    await vi.advanceTimersByTimeAsync(100)
    store().setQuery('gol')
    await pause()

    expect(pending.map((p) => p.url)).toEqual([searchURL('gol')])
  })

  it('trims the query it sends', async () => {
    store().setQuery('  golang ')
    await pause()

    expect(pending.map((p) => p.url)).toEqual([searchURL('golang')])
  })

  it('ignores a slow reply to an older query', async () => {
    store().setQuery('old')
    await pause()
    store().setQuery('new')
    await pause()

    respond(searchURL('new'), json(200, results('content/new.md')))
    await vi.advanceTimersByTimeAsync(0)
    respond(searchURL('old'), json(200, results('content/old.md')))
    await vi.advanceTimersByTimeAsync(0)

    expect(store().results?.hits[0]?.path).toBe('content/new.md')
    expect(store().loading).toBe(false)
  })

  it('reports an error and drops stale results', async () => {
    store().setQuery('x')
    await pause()
    respond(
      searchURL('x'),
      json(500, { code: 'cannot_search', message: 'The content could not be searched.' }),
    )
    await vi.advanceTimersByTimeAsync(0)

    expect(store().error?.code).toBe('cannot_search')
    expect(store().results).toBeNull()
    expect(store().loading).toBe(false)
  })

  it('clears results and sends nothing when the box is emptied', async () => {
    store().setQuery('x')
    await pause()
    respond(searchURL('x'), json(200, results('content/a.md')))
    await vi.advanceTimersByTimeAsync(0)

    store().setQuery('   ')
    await pause()

    expect(store().results).toBeNull()
    expect(store().loading).toBe(false)
    expect(pending).toHaveLength(0)
  })

  it('discards a reply that arrives after the box was emptied', async () => {
    store().setQuery('x')
    await pause()
    store().setQuery('')
    respond(searchURL('x'), json(200, results('content/a.md')))
    await vi.advanceTimersByTimeAsync(0)

    expect(store().results).toBeNull()
  })
})

describe('clear', () => {
  it('cancels a search that has not been sent yet', async () => {
    store().setQuery('x')
    store().clear()
    await pause()

    expect(pending).toHaveLength(0)
    expect(store().query).toBe('')
  })

  it('happens when a different project is opened', () => {
    store().setQuery('x')

    useProjectStore.setState({ project: null })
    useProjectStore.setState({
      project: {
        root: '/tmp/other',
        kind: 'hugo',
        configFile: 'hugo.toml',
        contentDir: 'content',
      contentDirs: [{ path: 'content' }],
        assetDir: 'static',
        id: 'other',
      },
    })

    expect(store().query).toBe('')
  })
})
