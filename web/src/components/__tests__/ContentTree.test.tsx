import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import type { ContentNode } from '@/api/types'
import { ContentTree } from '../ContentTree'

const file = (name: string, fields?: ContentNode['fields']): ContentNode => ({
  name,
  path: `content/${name}`,
  isDir: false,
  fields,
})

function renderNode(node: ContentNode, onSelect = vi.fn<(path: string) => void>()) {
  render(
    <ul>
      <ContentTree node={node} selected={null} onSelect={onSelect} />
    </ul>,
  )
  return onSelect
}

describe('ContentTree file row', () => {
  it('shows the title, the filename and the draft badge', () => {
    renderNode(
      file('hello.md', { title: 'Hello World', draft: true, date: '2024-03-01T00:00:00Z' }),
    )

    expect(screen.getByText('Hello World')).toBeInTheDocument()
    expect(screen.getByText(/hello\.md/)).toBeInTheDocument()
    expect(screen.getByText('Draft')).toBeInTheDocument()
  })

  it('falls back to the filename without frontmatter', () => {
    renderNode(file('plain.md'))

    expect(screen.getByText('plain.md')).toBeInTheDocument()
    expect(screen.queryByText('Draft')).not.toBeInTheDocument()
  })

  it('falls back to the filename when the title is empty', () => {
    renderNode(file('untitled.md', { title: '', draft: false }))

    expect(screen.getByText('untitled.md')).toBeInTheDocument()
  })

  it('selects the page by path on click', async () => {
    const onSelect = renderNode(file('hello.md', { title: 'Hello', draft: false }))

    await userEvent.click(screen.getByRole('button'))

    expect(onSelect).toHaveBeenCalledWith('content/hello.md')
  })
})
