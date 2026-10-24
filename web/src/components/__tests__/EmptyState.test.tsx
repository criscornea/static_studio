import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FileText } from 'lucide-react'
import { describe, expect, it, vi } from 'vitest'

import { useBuildStore } from '@/stores/build'
import { BuildPanel } from '../BuildPanel'
import { EmptyState, SkeletonRows } from '../EmptyState'

describe('EmptyState', () => {
  it('shows the text and runs the action', async () => {
    const onClick = vi.fn<() => void>()
    render(<EmptyState icon={FileText} text="Nothing here." action={{ label: 'Add', onClick }} />)
    expect(screen.getByText('Nothing here.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Add' }))
    expect(onClick).toHaveBeenCalledOnce()
  })
})

describe('SkeletonRows', () => {
  it('announces what is loading', () => {
    render(<SkeletonRows label="Loading pages" />)
    expect(screen.getByRole('status')).toHaveTextContent('Loading pages')
  })
})

describe('BuildPanel empty state', () => {
  it('offers a build when there is no output', () => {
    useBuildStore.setState({ status: 'idle', lines: [], problems: [], error: null })
    render(<BuildPanel />)
    expect(screen.getByText('No build output yet.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Build site' })).toBeInTheDocument()
  })
})
