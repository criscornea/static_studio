import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { EditorHeader } from '@/components/EditorHeader'
import { useBuildStore } from '@/stores/build'
import { useDevServerStore } from '@/stores/devserver'
import { useLayoutStore } from '@/stores/layout'
import { usePageStore } from '@/stores/page'

afterEach(() => {
  usePageStore.getState().clear()
  useDevServerStore.getState().reset()
  useBuildStore.getState().reset()
  useLayoutStore.setState({ previewOpen: false, buildLogOpen: false })
  vi.restoreAllMocks()
})

describe('EditorHeader', () => {
  it('shows the page path as a breadcrumb', () => {
    usePageStore.setState({
      path: 'content/posts/a.md',
      page: { path: 'content/posts/a.md' } as never,
      dirty: true,
    })
    render(<EditorHeader />)
    const crumbs = screen.getByRole('navigation', { name: 'Breadcrumb' })
    expect(crumbs).toHaveTextContent('contentpostsa.md')
    expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('offers the icon actions with accessible names', () => {
    render(<EditorHeader />)
    for (const name of ['Preview', 'Build', 'Publish']) {
      expect(screen.getByRole('button', { name })).toBeInTheDocument()
    }
    expect(screen.getByRole('button', { name: 'Preview' })).toBeDisabled()
  })

  it('toggles the preview once the server is ready', async () => {
    useDevServerStore.setState({ running: true, ready: true })
    render(<EditorHeader />)
    await userEvent.click(screen.getByRole('button', { name: 'Preview' }))
    expect(useLayoutStore.getState().previewOpen).toBe(true)
    expect(screen.getByRole('button', { name: 'Hide preview' })).toBeInTheDocument()
  })

  it('opens the build log when a build starts', async () => {
    const start = vi.fn<() => Promise<void>>().mockResolvedValue(undefined)
    useBuildStore.setState({ start })
    render(<EditorHeader />)
    await userEvent.click(screen.getByRole('button', { name: 'Build' }))
    expect(start).toHaveBeenCalled()
    expect(useLayoutStore.getState().buildLogOpen).toBe(true)
  })
})
