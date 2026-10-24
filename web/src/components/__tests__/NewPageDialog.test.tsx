import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ContentNode, Page } from '@/api/types'
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'
import { NewPageDialog } from '../NewPageDialog'

const tree: ContentNode = {
  name: 'content',
  path: 'content',
  isDir: true,
  children: [
    {
      name: 'posts',
      path: 'content/posts',
      isDir: true,
      children: [{ name: 'a.md', path: 'content/posts/a.md', isDir: false }],
    },
  ],
}

const created: Page = {
  path: 'content/posts/hello-world.md',
  format: 'toml',
  fields: { title: 'Hello World', draft: true },
  body: '',
  hash: 'h',
}

interface Call {
  method: string
  url: string
  body: unknown
}

let calls: Call[] = []

/** Answers by method and path; `createPage` decides the POST /api/page reply. */
function stubBackend(createPage: () => Response) {
  vi.stubGlobal('fetch', (input: RequestInfo, init?: RequestInit) => {
    const url = String(input)
    const method = init?.method ?? 'GET'
    const body = typeof init?.body === 'string' ? (JSON.parse(init.body) as unknown) : null
    calls.push({ method, url, body })

    const json = (status: number, payload: unknown) =>
      Promise.resolve(new Response(JSON.stringify(payload), { status }))

    if (url === '/api/archetypes') {
      return json(200, { archetypes: [{ name: 'default', builtin: true }, { name: 'posts' }] })
    }
    if (url === '/api/page' && method === 'POST') {
      return Promise.resolve(createPage())
    }
    if (url === '/api/content') {
      return json(200, { root: tree })
    }
    if (url.startsWith('/api/page?')) {
      return json(200, created)
    }
    throw new Error(`unexpected request: ${method} ${url}`)
  })
}

beforeEach(() => {
  calls = []
  useProjectStore.setState({
    project: {
      root: '/site',
      kind: 'hugo',
      configFile: 'hugo.toml',
      contentDir: 'content',
      assetDir: 'static',
      id: 'p1',
    },
    tree,
  })
  usePageStore.getState().clear()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('NewPageDialog', () => {
  it('creates the page, reloads the tree and opens the new page', async () => {
    stubBackend(() => new Response(JSON.stringify(created), { status: 201 }))
    const onOpenChange = vi.fn<(open: boolean) => void>()
    render(<NewPageDialog open onOpenChange={onOpenChange} />)

    await userEvent.type(screen.getByLabelText('Name'), 'Hello World')
    await userEvent.type(screen.getByLabelText('Folder'), 'posts')
    await userEvent.click(screen.getByRole('button', { name: 'Create page' }))

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false))

    expect(calls.find((c) => c.method === 'POST')?.body).toEqual({
      name: 'Hello World',
      folder: 'posts',
      archetype: '',
    })
    expect(calls.some((c) => c.url === '/api/content')).toBe(true)
    expect(usePageStore.getState().path).toBe('content/posts/hello-world.md')
  })

  it('starts in the folder of the open page', async () => {
    usePageStore.setState({ path: 'content/posts/a.md' })
    stubBackend(() => new Response(null, { status: 500 }))
    render(<NewPageDialog open onOpenChange={() => {}} />)

    expect(screen.getByLabelText('Folder')).toHaveValue('posts')
    expect(await screen.findByLabelText('Template')).toBeInTheDocument()
  })

  it('needs a name before it can create', () => {
    stubBackend(() => new Response(null, { status: 500 }))
    render(<NewPageDialog open onOpenChange={() => {}} />)

    expect(screen.getByRole('button', { name: 'Create page' })).toBeDisabled()
  })

  it('shows the backend message and stays open when creating fails', async () => {
    stubBackend(
      () =>
        new Response(
          JSON.stringify({
            code: 'page_exists',
            message: 'A page with this name already exists in that folder.',
          }),
          { status: 409 },
        ),
    )
    const onOpenChange = vi.fn<(open: boolean) => void>()
    render(<NewPageDialog open onOpenChange={onOpenChange} />)

    await userEvent.type(screen.getByLabelText('Name'), 'Taken')
    await userEvent.click(screen.getByRole('button', { name: 'Create page' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('already exists')
    expect(onOpenChange).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Create page' })).toBeEnabled()
    expect(usePageStore.getState().path).toBeNull()
  })

  it('renders nothing while closed', () => {
    stubBackend(() => new Response(null, { status: 500 }))
    render(<NewPageDialog open={false} onOpenChange={() => {}} />)

    expect(screen.queryByLabelText('Name')).not.toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })
})
