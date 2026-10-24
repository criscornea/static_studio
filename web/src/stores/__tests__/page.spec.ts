import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { usePageStore } from '../page'
import { useProjectStore } from '../project'
import type { Page, Project } from '@/api/types'

const store = () => usePageStore.getState()

const aProject: Project = {
  root: '/tmp/example-site',
  kind: 'hugo',
  configFile: 'hugo.toml',
  contentDir: 'content',
  assetDir: 'static',
  id: 'id-1',
}

function aPage(path: string, title: string): Page {
  return { path, format: 'yaml', fields: { title, draft: false }, body: `Body of ${title}\n`, hash: `hash-${title}` }
}

const pageURL = (path: string) => `/api/page?path=${encodeURIComponent(path)}`

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

/** Makes fetch return promises that settle only when the test calls respond. */
function holdRequests() {
  vi.stubGlobal(
    'fetch',
    (input: RequestInfo) =>
      new Promise<Response>((resolve) => {
        pending.push({ url: String(input), resolve })
      }),
  )
}

function respond(url: string, res: Response) {
  const i = pending.findIndex((p) => p.url === url)
  const [req] = i < 0 ? [] : pending.splice(i, 1)

  if (req === undefined) {
    throw new Error(`no pending request for ${url}`)
  }
  req.resolve(res)
}

beforeEach(() => {
  useProjectStore.setState({ project: null })
  usePageStore.getState().clear()
  pending = []
  holdRequests()
  useProjectStore.setState({ project: aProject })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('load', () => {
  it('sets the path at once and the page when it arrives', async () => {

    const done = store().load('content/a.md')

    expect(store().path).toBe('content/a.md')
    expect(store().loading).toBe(true)
    expect(store().page).toBeNull()

    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await done

    expect(store().page?.fields.title).toBe('A')
    expect(store().loading).toBe(false)
    expect(store().error).toBeNull()
  })

  it('reports an error and shows no page', async () => {

    const done = store().load('content/broken.md')
    respond(
      pageURL('content/broken.md'),
      json(422, { code: 'invalid_frontmatter', message: 'The settings block could not be read.' }),
    )
    await done

    expect(store().page).toBeNull()
    expect(store().error?.code).toBe('invalid_frontmatter')
    expect(store().loading).toBe(false)
  })
})

describe('races', () => {
  it('shows the last clicked page even if an earlier response arrives later', async () => {

    const first = store().load('content/a.md')
    const second = store().load('content/b.md')

    respond(pageURL('content/b.md'), json(200, aPage('content/b.md', 'B')))
    await second
    expect(store().page?.fields.title).toBe('B')

    // The slow response for the first click lands last and must be ignored.
    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await first

    expect(store().path).toBe('content/b.md')
    expect(store().page?.fields.title).toBe('B')
  })

  it('stays loading while a newer request is still in flight', async () => {

    const first = store().load('content/a.md')
    const second = store().load('content/b.md')

    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await first

    expect(store().loading).toBe(true)
    expect(store().page).toBeNull()

    respond(pageURL('content/b.md'), json(200, aPage('content/b.md', 'B')))
    await second
    expect(store().loading).toBe(false)
  })

  it('does not let a stale error replace a newer page', async () => {

    const first = store().load('content/a.md')
    const second = store().load('content/b.md')

    respond(pageURL('content/b.md'), json(200, aPage('content/b.md', 'B')))
    await second
    respond(
      pageURL('content/a.md'),
      json(404, { code: 'page_not_found', message: 'This page no longer exists.' }),
    )
    await first

    expect(store().page?.fields.title).toBe('B')
    expect(store().error).toBeNull()
  })
})

describe('clearing', () => {
  it('discards a request that was in flight when cleared', async () => {

    const done = store().load('content/a.md')
    store().clear()
    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await done

    expect(store().page).toBeNull()
    expect(store().path).toBeNull()
    expect(store().loading).toBe(false)
  })

  it('clears itself when a different project is opened', async () => {

    const done = store().load('content/a.md')
    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await done
    expect(store().page).not.toBeNull()

    useProjectStore.setState({ project: { ...aProject, id: 'id-2' } })

    expect(store().page).toBeNull()
    expect(store().path).toBeNull()
  })

  it('clears itself when the project is closed', async () => {

    const done = store().load('content/a.md')
    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await done

    useProjectStore.setState({ project: null })

    expect(store().page).toBeNull()
  })
})

describe('save', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    store().clear()
  })

  it('sends the hash and adopts the new one', async () => {
    const calls: { url: string; init?: RequestInit }[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string, init?: RequestInit) => {
        calls.push({ url, init })
        if (init?.method === 'PUT') {
          return Promise.resolve(json(200, { hash: 'h2' }))
        }
        return Promise.resolve(json(200, aPage('content/a.md', 'A')))
      }),
    )
    await store().load('content/a.md')
    store().setDraft('edited')
    await store().save()

    expect(JSON.parse(calls[1]!.init?.body as string)).toEqual({
      path: 'content/a.md',
      body: 'edited',
      hash: 'hash-A',
    })
    expect(store().page).toMatchObject({ body: 'edited', hash: 'h2' })
    expect(store().saveError).toBeNull()
  })

  it('keeps the draft and reports a conflict', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((_url: string, init?: RequestInit) =>
        Promise.resolve(
          init?.method === 'PUT'
            ? json(409, { code: 'page_conflict', message: 'changed outside' })
            : json(200, aPage('content/a.md', 'A')),
        ),
      ),
    )
    await store().load('content/a.md')
    store().setDraft('edited')
    await store().save()

    expect(store().saveError?.code).toBe('page_conflict')
    expect(store().draft).toBe('edited')
    expect(store().page?.hash).toBe('hash-A')
    expect(store().savingPath).toBeNull()
  })
})
