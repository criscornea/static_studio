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
  return { path, format: 'yaml', fields: { title, draft: false }, body: `Body of ${title}\n`, bodyLine: 1, hash: `hash-${title}` }
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

describe('page store frontmatter fields', () => {
  it('sends changed fields on save and adopts them', async () => {
    const page = aPage('content/a.md', 'A')
    usePageStore.setState({ path: page.path, page, draft: page.body, fieldsDraft: page.fields })
    const bodies: unknown[] = []
    vi.stubGlobal('fetch', (_url: string, init?: RequestInit) => {
      bodies.push(JSON.parse(init?.body as string))
      return Promise.resolve(json(200, { hash: 'h2' }))
    })

    store().setFieldsDraft({ ...page.fields, title: 'B' })
    await store().save()
    expect(bodies[0]).toMatchObject({ fields: { title: 'B' }, hash: 'hash-A' })
    expect(store().page?.fields.title).toBe('B')
    expect(store().page?.hash).toBe('h2')

    // Unchanged fields are not sent again.
    store().setDraft('edited body')
    await store().save()
    expect((bodies[1] as { fields?: unknown }).fields).toBeUndefined()
  })
})

describe('autosave and dirty state', () => {
  const puts: unknown[] = []
  let putStatus = 200

  beforeEach(() => {
    vi.useFakeTimers()
    puts.length = 0
    putStatus = 200
    vi.stubGlobal(
      'fetch',
      vi.fn((_url: string, init?: RequestInit) => {
        if (init?.method === 'PUT') {
          puts.push(JSON.parse(init.body as string))
          return Promise.resolve(
            putStatus === 200
              ? json(200, { hash: `h${puts.length}` })
              : json(409, { code: 'page_conflict', message: 'changed outside' }),
          )
        }
        return Promise.resolve(json(200, aPage('content/a.md', 'A')))
      }),
    )
  })

  afterEach(() => {
    vi.useRealTimers()
    store().clear()
  })

  it('tracks dirty and clears it when the draft matches again', async () => {
    await store().load('content/a.md')
    expect(store().dirty).toBe(false)
    store().setDraft('x')
    expect(store().dirty).toBe(true)
    store().setDraft(store().page!.body)
    expect(store().dirty).toBe(false)
  })

  it('debounces edits into one save and adopts the new hash', async () => {
    await store().load('content/a.md')
    store().setDraft('x')
    await vi.advanceTimersByTimeAsync(1000)
    store().setDraft('xy')
    await vi.advanceTimersByTimeAsync(1000)
    expect(puts).toHaveLength(0)
    await vi.advanceTimersByTimeAsync(600)
    expect(puts).toEqual([{ path: 'content/a.md', body: 'xy', hash: 'hash-A' }])
    expect(store().page?.hash).toBe('h1')
    expect(store().dirty).toBe(false)
  })

  it('saves edits made during a save afterwards with the new hash', async () => {
    await store().load('content/a.md')
    store().setDraft('x')
    const first = store().save()
    await vi.advanceTimersByTimeAsync(0)
    store().setDraft('xy')
    await first
    await vi.advanceTimersByTimeAsync(2000)
    expect(puts).toMatchObject([{ body: 'x', hash: 'hash-A' }, { body: 'xy', hash: 'h1' }])
    expect(store().dirty).toBe(false)
  })

  it('flushes pending edits when switching pages', async () => {
    await store().load('content/a.md')
    store().setDraft('x')
    await store().load('content/a.md')
    expect(puts).toHaveLength(1)
  })

  it('stops autosave on a conflict and keeps the draft', async () => {
    await store().load('content/a.md')
    putStatus = 409
    store().setDraft('x')
    await vi.advanceTimersByTimeAsync(2000)
    expect(store().saveError?.code).toBe('page_conflict')
    expect(store().dirty).toBe(true)
    store().setDraft('xy')
    await vi.advanceTimersByTimeAsync(5000)
    expect(puts).toHaveLength(1)
    expect(store().draft).toBe('xy')
  })

  it('does not retry a failed non-conflict save on a timer', async () => {
    await store().load('content/a.md')
    putStatus = 500
    store().setDraft('x')
    await vi.advanceTimersByTimeAsync(2000)
    expect(puts).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(10000)
    expect(puts).toHaveLength(1)
    expect(store().dirty).toBe(true)
  })

  it('saves unsaved edits before the project closes, and aborts if the user cancels', async () => {
    await store().load('content/a.md')
    putStatus = 500
    store().setDraft('x')
    const confirm = vi.fn<() => boolean>(() => false)
    vi.stubGlobal('confirm', confirm)
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(json(500, { code: 'cannot_save_page', message: 'no' }))),
    )
    await useProjectStore.getState().close()
    expect(confirm).toHaveBeenCalled()
    expect(useProjectStore.getState().project).not.toBeNull()
    expect(store().draft).toBe('x')
  })

  it('makes the old page uneditable while a new one loads', async () => {
    await store().load('content/a.md')
    holdRequests()
    const p = store().load('content/b.md')
    expect(store().page).toBeNull()
    expect(store().draft).toBeNull()
    respond(pageURL('content/b.md'), json(200, aPage('content/b.md', 'B')))
    await p
    expect(store().page?.path).toBe('content/b.md')
  })
})
