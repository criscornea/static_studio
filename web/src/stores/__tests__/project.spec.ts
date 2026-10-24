import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { selectIsOpen, useProjectStore } from '../project'
import { setProjectID } from '@/api/client'
import type { ContentTree, Project } from '@/api/types'

const projectPath = '/tmp/example-site'

const store = () => useProjectStore.getState()

const aProject: Project = {
  root: projectPath,
  kind: 'hugo',
  configFile: 'hugo.toml',
  contentDir: 'content',
      contentDirs: [{ path: 'content' }],
  assetDir: 'static',
  id: 'id-1',
}

const aRoot = {
  name: 'content',
  path: 'content',
  isDir: true,
  children: [{ name: 'about.md', path: 'content/about.md', isDir: false, ext: '.md', size: 12 }],
}

const aTree: ContentTree = { root: aRoot, roots: [{ dir: 'content', root: aRoot }] }

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
  useProjectStore.setState({ project: null, tree: null, loading: false, error: null })
  setProjectID(null)
  calls = []
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('open', () => {
  it('stores the project, loads the tree and sends the id', async () => {
    respondWith({ status: 200, body: aProject }, { status: 200, body: aTree })

    const ok = await store().open(projectPath)

    expect(ok).toBe(true)
    expect(selectIsOpen(store())).toBe(true)
    expect(store().project?.id).toBe('id-1')
    expect(store().tree?.children).toHaveLength(1)
    expect(store().roots.map((r) => r.dir)).toEqual(["content"])
    expect(store().error).toBeNull()
    expect(store().loading).toBe(false)

    expect(call(0).url).toBe('/api/project/open')
    expect(call(0).method).toBe('POST')
    expect(call(0).body).toBe(JSON.stringify({ path: projectPath }))

    // The tree request must carry the id returned by open.
    expect(call(1).url).toBe('/api/content')
    expect(call(1).headers.get('X-Project-ID')).toBe('id-1')
  })

  it('reports a backend error and stays closed', async () => {
    respondWith({
      status: 422,
      body: { code: 'not_a_project', message: "This folder doesn't look like a website project." },
    })

    const ok = await store().open('/tmp')

    expect(ok).toBe(false)
    expect(selectIsOpen(store())).toBe(false)
    expect(store().tree).toBeNull()
    expect(store().error?.code).toBe('not_a_project')
    expect(store().error?.message).toContain('website project')
  })

  it('does not stay half-open when the tree fails to load', async () => {
    respondWith(
      { status: 200, body: aProject },
      { status: 500, body: { code: 'cannot_read_content', message: 'Could not read content.' } },
    )

    const ok = await store().open(projectPath)

    expect(ok).toBe(false)
    expect(store().project).toBeNull()
    expect(store().tree).toBeNull()
    expect(store().error?.code).toBe('cannot_read_content')
  })

  it('turns a network failure into a readable error', async () => {
    vi.stubGlobal('fetch', () => Promise.reject(new TypeError('Failed to fetch')))


    expect(await store().open(projectPath)).toBe(false)
    expect(store().error?.status).toBe(0)
    expect(store().error?.message).toContain('static_studio')
  })
})

describe('restore', () => {
  it('reconnects to the project the backend already has open', async () => {
    respondWith({ status: 200, body: aProject }, { status: 200, body: aTree })

    await store().restore()

    expect(selectIsOpen(store())).toBe(true)
    expect(call(0).url).toBe('/api/project')
  })

  it('treats no_project as the normal cold start, not an error', async () => {
    respondWith({ status: 409, body: { code: 'no_project', message: 'No project is open.' } })

    await store().restore()

    expect(selectIsOpen(store())).toBe(false)
    expect(store().error).toBeNull()
  })

  it('surfaces any other failure', async () => {
    respondWith({ status: 500, body: { code: 'internal', message: 'Something went wrong.' } })

    await store().restore()

    expect(store().error?.code).toBe('internal')
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

    await store().open(projectPath)
    await store().close()

    expect(selectIsOpen(store())).toBe(false)
    expect(store().tree).toBeNull()

    // A later request must not carry the old id.
    await store().restore()
    expect(call(3).headers.get('X-Project-ID')).toBeNull()
  })
})

function call(i: number): Call {
  const c = calls[i]
  if (c === undefined) {
    throw new Error(`expected at least ${i + 1} fetch calls, got ${calls.length}`)
  }
  return c
}
