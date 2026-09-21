import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { usePageStore } from '../page'
import { useProjectStore } from '../project'
import type { Page, Project } from '@/api/types'

const aProject: Project = {
  root: '/tmp/example-site',
  kind: 'hugo',
  configFile: 'hugo.toml',
  contentDir: 'content',
  assetDir: 'static',
  id: 'id-1',
}

function aPage(path: string, title: string): Page {
  return { path, format: 'yaml', fields: { title, draft: false }, body: `Body of ${title}\n` }
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
  setActivePinia(createPinia())
  pending = []
  holdRequests()
  useProjectStore().project = aProject
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('load', () => {
  it('sets the path at once and the page when it arrives', async () => {
    const store = usePageStore()

    const done = store.load('content/a.md')

    expect(store.path).toBe('content/a.md')
    expect(store.loading).toBe(true)
    expect(store.page).toBeNull()

    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await done

    expect(store.page?.fields.title).toBe('A')
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('reports an error and shows no page', async () => {
    const store = usePageStore()

    const done = store.load('content/broken.md')
    respond(
      pageURL('content/broken.md'),
      json(422, { code: 'invalid_frontmatter', message: 'The settings block could not be read.' }),
    )
    await done

    expect(store.page).toBeNull()
    expect(store.error?.code).toBe('invalid_frontmatter')
    expect(store.loading).toBe(false)
  })
})

describe('races', () => {
  it('shows the last clicked page even if an earlier response arrives later', async () => {
    const store = usePageStore()

    const first = store.load('content/a.md')
    const second = store.load('content/b.md')

    respond(pageURL('content/b.md'), json(200, aPage('content/b.md', 'B')))
    await second
    expect(store.page?.fields.title).toBe('B')

    // The slow response for the first click lands last and must be ignored.
    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await first

    expect(store.path).toBe('content/b.md')
    expect(store.page?.fields.title).toBe('B')
  })

  it('stays loading while a newer request is still in flight', async () => {
    const store = usePageStore()

    const first = store.load('content/a.md')
    const second = store.load('content/b.md')

    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await first

    expect(store.loading).toBe(true)
    expect(store.page).toBeNull()

    respond(pageURL('content/b.md'), json(200, aPage('content/b.md', 'B')))
    await second
    expect(store.loading).toBe(false)
  })

  it('does not let a stale error replace a newer page', async () => {
    const store = usePageStore()

    const first = store.load('content/a.md')
    const second = store.load('content/b.md')

    respond(pageURL('content/b.md'), json(200, aPage('content/b.md', 'B')))
    await second
    respond(
      pageURL('content/a.md'),
      json(404, { code: 'page_not_found', message: 'This page no longer exists.' }),
    )
    await first

    expect(store.page?.fields.title).toBe('B')
    expect(store.error).toBeNull()
  })
})

describe('clearing', () => {
  it('discards a request that was in flight when cleared', async () => {
    const store = usePageStore()

    const done = store.load('content/a.md')
    store.clear()
    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await done

    expect(store.page).toBeNull()
    expect(store.path).toBeNull()
    expect(store.loading).toBe(false)
  })

  it('clears itself when a different project is opened', async () => {
    const store = usePageStore()

    const done = store.load('content/a.md')
    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await done
    expect(store.page).not.toBeNull()

    useProjectStore().project = { ...aProject, id: 'id-2' }
    await nextTick()

    expect(store.page).toBeNull()
    expect(store.path).toBeNull()
  })

  it('clears itself when the project is closed', async () => {
    const store = usePageStore()

    const done = store.load('content/a.md')
    respond(pageURL('content/a.md'), json(200, aPage('content/a.md', 'A')))
    await done

    useProjectStore().project = null
    await nextTick()

    expect(store.page).toBeNull()
  })
})
