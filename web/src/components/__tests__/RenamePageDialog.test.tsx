import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ContentNode } from '@/api/types'
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'
import { RenamePageDialog } from '../RenamePageDialog'

const tree: ContentNode = { name: 'content', path: 'content', isDir: true, children: [] }

interface Call {
  method: string
  url: string
  body: unknown
}

let calls: Call[] = []

function stubBackend(rename: () => Response) {
  vi.stubGlobal('fetch', (input: RequestInfo, init?: RequestInit) => {
    const url = String(input)
    const method = init?.method ?? 'GET'
    const body = typeof init?.body === 'string' ? (JSON.parse(init.body) as unknown) : null
    calls.push({ method, url, body })
    const json = (status: number, payload: unknown) =>
      Promise.resolve(new Response(JSON.stringify(payload), { status }))

    if (url === '/api/page/rename') {
      return Promise.resolve(rename())
    }
    if (url === '/api/content') {
      return json(200, { root: tree })
    }
    if (url.startsWith('/api/page?')) {
      return json(200, { path: 'content/posts/b.md', format: '', fields: { title: '', draft: false }, body: '' })
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

describe('RenamePageDialog', () => {
  it('renames, reloads the tree and follows the open page', async () => {
    stubBackend(() => new Response(JSON.stringify({ path: 'content/posts/b.md' }), { status: 200 }))
    usePageStore.setState({ path: 'content/a.md' })
    const onClose = vi.fn<() => void>()
    render(<RenamePageDialog path="content/a.md" onClose={onClose} />)

    const input = screen.getByLabelText(/Path inside/)
    expect(input).toHaveValue('a.md')
    expect(screen.getByRole('button', { name: 'Rename' })).toBeDisabled()

    await userEvent.clear(input)
    await userEvent.type(input, 'posts/b.md')
    await userEvent.click(screen.getByRole('button', { name: 'Rename' }))

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(calls.find((c) => c.method === 'POST')?.body).toEqual({
      from: 'content/a.md',
      to: 'content/posts/b.md',
    })
    expect(calls.some((c) => c.url === '/api/content')).toBe(true)
    expect(usePageStore.getState().path).toBe('content/posts/b.md')
  })

  it('shows the backend message and stays open on refusal', async () => {
    stubBackend(
      () =>
        new Response(
          JSON.stringify({ code: 'page_exists', message: 'A page with this name already exists.' }),
          { status: 409 },
        ),
    )
    const onClose = vi.fn<() => void>()
    render(<RenamePageDialog path="content/a.md" onClose={onClose} />)

    await userEvent.type(screen.getByLabelText(/Path inside/), 'x')
    await userEvent.click(screen.getByRole('button', { name: 'Rename' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('already exists')
    expect(onClose).not.toHaveBeenCalled()
  })
})
