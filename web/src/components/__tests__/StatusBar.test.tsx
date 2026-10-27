import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { StatusBar } from '@/components/StatusBar'
import userEvent from '@testing-library/user-event'
import { useLayoutStore } from '@/stores/layout'
import { useBuildStore } from '@/stores/build'
import { useDevServerStore } from '@/stores/devserver'
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'
import type { Project } from '@/api/types'

afterEach(() => {
  useProjectStore.setState({ project: null })
  usePageStore.setState({ page: null, dirty: false, savingPath: null, saveError: null })
  useDevServerStore.getState().reset()
  useBuildStore.getState().reset()
})

describe('StatusBar', () => {
  it('shows the project name and idle states', () => {
    useProjectStore.setState({ project: { root: '/home/me/site' } as Project })
    render(<StatusBar />)
    expect(screen.getByText('site')).toHaveClass('font-semibold')
    expect(screen.getByTitle('/home/me/site')).toHaveTextContent('/home/me/site')
    expect(screen.getByText('Server: stopped')).toBeInTheDocument()
    expect(screen.getByText('Build: idle')).toBeInTheDocument()
  })

  it('reflects save, preview and build state', () => {
    usePageStore.setState({ page: { path: 'content/a.md' } as never, dirty: true })
    useDevServerStore.setState({ running: true, ready: true, url: 'http://127.0.0.1:4000' })
    useBuildStore.setState({ status: 'failed' })
    render(<StatusBar />)
    expect(screen.getByText('Unsaved changes')).toBeInTheDocument()
    expect(screen.getByText('Server: running :4000')).toBeInTheDocument()
    expect(screen.getByText('Build: failed')).toBeInTheDocument()
  })
})

describe('StatusBar path', () => {
  it('bolds the last segment for both separators and trailing ones', () => {
    useProjectStore.setState({ project: { root: 'C:\\sites\\blog\\' } as Project })
    render(<StatusBar />)
    expect(screen.getByText('blog')).toHaveClass('font-semibold')
    expect(screen.getByTitle('C:\\sites\\blog\\')).toHaveTextContent('C:\\sites\\blog')
  })
})

describe('StatusBar build log toggle', () => {
  it('toggles the build log panel', async () => {
    useLayoutStore.setState({ buildLogOpen: false })
    render(<StatusBar />)
    const toggle = screen.getByRole('button', { name: /Build:/ })
    expect(toggle).toHaveAttribute('aria-pressed', 'false')
    await userEvent.click(toggle)
    expect(useLayoutStore.getState().buildLogOpen).toBe(true)
    expect(toggle).toHaveAttribute('aria-pressed', 'true')
    useLayoutStore.setState({ buildLogOpen: false })
  })
})
