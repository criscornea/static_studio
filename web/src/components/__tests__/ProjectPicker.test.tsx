import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ProjectPicker } from '../ProjectPicker'
import { api } from '@/api/client'
import { useProjectStore } from '@/stores/project'

const entries = [
  { path: '/sites/blog', name: 'blog', kind: 'hugo', openedAt: '2026-01-01T00:00:00Z', available: true },
  { path: '/sites/gone', name: 'gone', kind: 'hugo', openedAt: '2026-01-01T00:00:00Z', available: false },
] as const

beforeEach(() => {
  vi.spyOn(api, 'recentProjects').mockResolvedValue([...entries])
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ProjectPicker recent list', () => {
  it('opens an available entry on click', async () => {
    const open = vi.fn<(path: string) => Promise<boolean>>().mockResolvedValue(true)
    useProjectStore.setState({ open })
    render(<ProjectPicker />)
    await userEvent.click(await screen.findByRole('button', { name: /^blog/ }))
    expect(open).toHaveBeenCalledWith('/sites/blog')
  })

  it('shows unavailable entries greyed with only a remove action', async () => {
    render(<ProjectPicker />)
    await screen.findByText('Folder not found')
    expect(screen.queryByRole('button', { name: /^gone/ })).toBeNull()
    expect(screen.getByRole('button', { name: /Remove gone/ })).toBeInTheDocument()
  })

  it('removes an entry', async () => {
    const remove = vi.spyOn(api, 'removeRecent').mockResolvedValue(undefined)
    render(<ProjectPicker />)
    await userEvent.click(await screen.findByRole('button', { name: /Remove blog/ }))
    expect(remove).toHaveBeenCalledWith('/sites/blog')
    await waitFor(() => expect(screen.queryByText('/sites/blog')).toBeNull())
  })
})
