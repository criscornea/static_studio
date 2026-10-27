import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import App from '@/App'
import { useProjectStore } from '@/stores/project'
import type { Project } from '@/api/types'

afterEach(() => {
  useProjectStore.setState({ project: null })
})

describe('App shell', () => {
  it('puts Options and Close in the activity bar, with no theme button or top bar', () => {
    useProjectStore.setState({ project: { root: '/home/me/site' } as Project, restore: vi.fn<() => Promise<void>>() })
    const { container } = render(<App />)
    const nav = screen.getByRole('navigation', { name: 'Activity bar' })
    const buttons = Array.from(nav.querySelectorAll('button'))
    expect(buttons.at(-1)).toHaveAttribute('aria-label', 'Close')
    expect(buttons.at(-2)).toHaveAttribute('aria-label', 'Options')
    expect(screen.queryByRole('button', { name: /Theme/ })).toBeNull()
    expect(container.querySelector('header')).toBeNull()
  })
})
