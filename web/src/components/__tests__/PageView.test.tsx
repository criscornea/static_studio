import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/api/client'
import type { Page } from '@/api/types'
import { useDevServerStore } from '@/stores/devserver'
import { useLayoutStore } from '@/stores/layout'
import { usePageStore } from '@/stores/page'
import { PageView } from '../PageView'

const page = {
  path: 'content/a.md',
  body: 'hello',
  hash: 'h1',
  bodyLine: 1,
  fields: { title: 'Hi', draft: false },
} as unknown as Page

beforeEach(() => {
  vi.spyOn(api, 'page').mockResolvedValue(page)
  vi.spyOn(api, 'savePage').mockResolvedValue({ hash: 'h2' })
  usePageStore.getState().clear()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('PageView', () => {
  it('keeps a single form and unique keys while typing', async () => {
    const errors = vi.spyOn(console, 'error').mockImplementation(() => {})
    await usePageStore.getState().load('content/a.md')
    const { container } = render(<PageView />)
    await userEvent.type(screen.getByLabelText(/title/i), 'fg')
    expect(container.querySelectorAll('form')).toHaveLength(1)
    const dup = errors.mock.calls.filter((c) => String(c[0]).includes('same key'))
    expect(dup).toHaveLength(0)
  })
})

describe('PageView preview pane', () => {
  afterEach(() => {
    useLayoutStore.setState({ previewOpen: false })
    useDevServerStore.getState().reset()
  })

  it('shows the preview beside the editor only while it is open', async () => {
    vi.spyOn(api, 'devServer').mockReturnValue(new Promise(() => {}))
    useDevServerStore.setState({ running: true, ready: true })
    await usePageStore.getState().load('content/a.md')
    render(<PageView />)
    expect(screen.queryByText(/Loading preview/)).not.toBeInTheDocument()
    act(() => useLayoutStore.getState().setPreviewOpen(true))
    expect(await screen.findByText(/Loading preview|no preview address/)).toBeInTheDocument()
    act(() => useLayoutStore.getState().setPreviewOpen(false))
    expect(screen.queryByText(/Loading preview|no preview address/)).not.toBeInTheDocument()
  })
})
