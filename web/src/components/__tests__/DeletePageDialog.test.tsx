import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ContentNode } from '@/api/types'
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'
import { DeletePageDialog } from '../DeletePageDialog'

const tree: ContentNode = { name: 'content', path: 'content', isDir: true, children: [] }

let calls: { method: string; url: string }[] = []

function stubBackend(del: () => Response) {
  vi.stubGlobal('fetch', (input: RequestInfo, init?: RequestInit) => {
    const url = String(input)
    const method = init?.method ?? 'GET'
    calls.push({ method, url })
    if (method === 'DELETE') {
      return Promise.resolve(del())
    }
    if (url.startsWith('/api/page?')) {
      return Promise.resolve(new Response(JSON.stringify({ hash: 'h1' }), { status: 200 }))
    }
    if (url === '/api/content') {
      return Promise.resolve(new Response(JSON.stringify({ root: tree }), { status: 200 }))
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
      contentDirs: [{ path: 'content' }],
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

describe('DeletePageDialog', () => {
  it('deletes only after confirming, then clears the open page', async () => {
    stubBackend(() => new Response(null, { status: 204 }))
    usePageStore.setState({ path: 'content/a.md' })
    const onClose = vi.fn<() => void>()
    render(<DeletePageDialog path="content/a.md" onClose={onClose} />)

    expect(screen.getByText('content/a.md')).toBeInTheDocument()
    expect(calls).toHaveLength(0)

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(calls[0]).toEqual({ method: 'GET', url: '/api/page?path=content%2Fa.md' })
    expect(calls[1]).toEqual({
      method: 'DELETE',
      url: '/api/page?path=content%2Fa.md&hash=h1',
    })
    expect(calls.some((c) => c.url === '/api/content')).toBe(true)
    expect(usePageStore.getState().path).toBeNull()
  })

  it('cancel deletes nothing', async () => {
    stubBackend(() => new Response(null, { status: 204 }))
    const onClose = vi.fn<() => void>()
    render(<DeletePageDialog path="content/a.md" onClose={onClose} />)

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(onClose).toHaveBeenCalled()
    expect(calls).toHaveLength(0)
  })

  it('keeps another open page and shows the refusal message', async () => {
    stubBackend(
      () =>
        new Response(JSON.stringify({ code: 'not_editable', message: 'Cannot delete this.' }), {
          status: 400,
        }),
    )
    usePageStore.setState({ path: 'content/other.md' })
    const onClose = vi.fn<() => void>()
    render(<DeletePageDialog path="content/a.md" onClose={onClose} />)

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Cannot delete this.')
    expect(onClose).not.toHaveBeenCalled()
    expect(usePageStore.getState().path).toBe('content/other.md')
  })
})
