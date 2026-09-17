import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useProjectStore } from '../project'
import type { ContentTree, Project } from '@/api/types'

const projectPath = '/tmp/example-site'

const aProject: Project = {
  root: projectPath,
  kind: 'hugo',
  configFile: 'hugo.toml',
  contentDir: 'content',
  assetDir: 'static',
  id: 'id-1',
}

const aTree: ContentTree = {
  root: {
    name: 'content',
    path: 'content',
    isDir: true,
    children: [{ name: 'about.md', path: 'content/about.md', isDir: false, ext: '.md', size: 12 }],
  },
}

/** One recorded fetch call, so tests can assert on headers. */
interface Call {
  url: string
  method: string
  headers: Headers
  body: string | null
}

let calls: Call[] = []

/** Queues responses to be returned in order by the fake fetch. */
function respondWith(...responses: Array<{ status: number; body?: unknown }>) {
  const queue = [...responses]

  vi.stubGlobal('fetch', (input: RequestInfo, init?: RequestInit) => {
    const next = queue.shift()
    if (next === undefined) {
      throw new Error(`unexpected fetch call: ${String(input)}`)
    }
    calls.push({
      url: String(input),
      method: init?.method ?? 'GET',
      headers: new Headers(init?.headers),
      body: typeof init?.body === 'string' ? init.body : null,
    })
    return Promise.resolve(
      new Response(next.body === undefined ? null : JSON.stringify(next.body), {
        status: next.status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

beforeEach(() => {
  setActivePinia(createPinia())
  calls = []
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('open', () => {
  it('stores the project, loads the tree and sends the id', async () => {
    respondWith({ status: 200, body: aProject }, { status: 200, body: aTree })

    const store = useProjectStore()
    const ok = await store.open(projectPath)

    expect(ok).toBe(true)
    expect(store.isOpen).toBe(true)
    expect(store.project?.id).toBe('id-1')
    expect(store.tree?.children).toHaveLength(1)
    expect(store.error).toBeNull()
    expect(store.loading).toBe(false)

    expect(calls[0].url).toBe('/api/project/open')
    expect(calls[0].method).toBe('POST')
    expect(calls[0].body).toBe(JSON.stringify({ path: projectPath }))

    // The tree request must carry the id returned by open.
    expect(calls[1].url).toBe('/api/content')
    expect(calls[1].headers.get('X-Project-ID')).toBe('id-1')
  })

  it('reports a backend error and stays closed', async () => {
    respondWith({
      status: 422,
      body: { code: 'not_a_project', message: "This folder doesn't look like a website project." },
    })

    const store = useProjectStore()
    const ok = await store.open('/tmp')

    expect(ok).toBe(false)
    expect(store.isOpen).toBe(false)
    expect(store.tree).toBeNull()
    expect(store.error?.code).toBe('not_a_project')
    expect(store.error?.message).toContain('website project')
  })

  it('does not stay half-open when the tree fails to load', async () => {
    respondWith(
      { status: 200, body: aProject },
      { status: 500, body: { code: 'cannot_read_content', message: 'Could not read content.' } },
    )

    const store = useProjectStore()
    const ok = await store.open(projectPath)

    expect(ok).toBe(false)
    expect(store.project).toBeNull()
    expect(store.tree).toBeNull()
    expect(store.error?.code).toBe('cannot_read_content')
  })

  it('turns a network failure into a readable error', async () => {
    vi.stubGlobal('fetch', () => Promise.reject(new TypeError('Failed to fetch')))

    const store = useProjectStore()

    expect(await store.open(projectPath)).toBe(false)
    expect(store.error?.status).toBe(0)
    expect(store.error?.message).toContain('static_studio')
  })
})

describe('restore', () => {
  it('reconnects to the project the backend already has open', async () => {
    respondWith({ status: 200, body: aProject }, { status: 200, body: aTree })

    const store = useProjectStore()
    await store.restore()

    expect(store.isOpen).toBe(true)
    expect(calls[0].url).toBe('/api/project')
  })

  it('treats no_project as the normal cold start, not an error', async () => {
    respondWith({ status: 409, body: { code: 'no_project', message: 'No project is open.' } })

    const store = useProjectStore()
    await store.restore()

    expect(store.isOpen).toBe(false)
    expect(store.error).toBeNull()
  })

  it('surfaces any other failure', async () => {
    respondWith({ status: 500, body: { code: 'internal', message: 'Something went wrong.' } })

    const store = useProjectStore()
    await store.restore()

    expect(store.error?.code).toBe('internal')
  })
})

describe('close', () => {
  it('clears the state and stops sending the id', async () => {
    respondWith(
      { status: 200, body: aProject },
      { status: 200, body: aTree },
      { status: 204 },
      { status: 409, body: { code: 'no_project', message: 'No project is open.' } },
    )

    const store = useProjectStore()
    await store.open(projectPath)
    await store.close()

    expect(store.isOpen).toBe(false)
    expect(store.tree).toBeNull()

    // A later request must not carry the old id.
    await store.restore()
    expect(calls[3].headers.get('X-Project-ID')).toBeNull()
  })
})
