import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import type { ContentNode } from '@/api/types'
import { ContentTree, TreeRoot } from '../ContentTree'

const file = (name: string, fields?: ContentNode['fields']): ContentNode => ({
  name,
  path: `content/${name}`,
  isDir: false,
  fields,
})

function renderNode(node: ContentNode, onSelect = vi.fn<(path: string) => void>()) {
  render(
    <TreeRoot label="Pages">
      <ContentTree node={node} selected={null} onSelect={onSelect} />
    </TreeRoot>,
  )
  return onSelect
}

describe('ContentTree file row', () => {
  it('shows the title and the draft marker', () => {
    renderNode(
      file('hello.md', { title: 'Hello World', draft: true, date: '2024-03-01T00:00:00Z' }),
    )

    expect(screen.getByText('Hello World')).toBeInTheDocument()
    expect(screen.getByRole('treeitem')).toHaveAttribute('title', 'content/hello.md')
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

    await userEvent.click(screen.getByRole('treeitem'))

    expect(onSelect).toHaveBeenCalledWith('content/hello.md')
  })
})

const folder: ContentNode = {
  name: 'posts',
  path: 'content/posts',
  isDir: true,
  children: [
    { name: 'a.md', path: 'content/posts/a.md', isDir: false },
    { name: 'b.md', path: 'content/posts/b.md', isDir: false },
  ],
}

function renderTree(handlers: Partial<Parameters<typeof ContentTree>[0]> = {}) {
  const onSelect = vi.fn<(path: string) => void>()
  render(
    <TreeRoot label="Pages">
      <ContentTree node={folder} selected={null} onSelect={onSelect} {...handlers} />
      <ContentTree node={file('z.md')} selected={null} onSelect={onSelect} {...handlers} />
    </TreeRoot>,
  )
  return onSelect
}

describe('ContentTree keyboard', () => {
  it('moves with arrows, Home and End, and opens with Enter', async () => {
    const onSelect = renderTree()
    const items = screen.getAllByRole('treeitem')
    items[0]?.focus()

    await userEvent.keyboard('{ArrowDown}')
    expect(screen.getByText('a.md').closest('[role=treeitem]')).toHaveFocus()

    await userEvent.keyboard('{End}')
    expect(screen.getByText('z.md').closest('[role=treeitem]')).toHaveFocus()

    await userEvent.keyboard('{ArrowUp}{ArrowUp}')
    expect(screen.getByText('a.md').closest('[role=treeitem]')).toHaveFocus()

    await userEvent.keyboard('{Home}')
    expect(items[0]).toHaveFocus()

    await userEvent.keyboard('{ArrowDown}{Enter}')
    expect(onSelect).toHaveBeenCalledWith('content/posts/a.md')
  })

  it('collapses and expands folders, skipping hidden rows', async () => {
    renderTree()
    const dir = screen.getByText('posts').closest<HTMLElement>('[role=treeitem]')
    dir?.focus()

    await userEvent.keyboard('{ArrowLeft}')
    expect(dir).toHaveAttribute('aria-expanded', 'false')
    await userEvent.keyboard('{ArrowDown}')
    expect(screen.getByText('z.md').closest('[role=treeitem]')).toHaveFocus()

    await userEvent.keyboard('{ArrowUp}{ArrowRight}')
    expect(dir).toHaveAttribute('aria-expanded', 'true')
    await userEvent.keyboard('{ArrowRight}')
    expect(screen.getByText('a.md').closest('[role=treeitem]')).toHaveFocus()

    await userEvent.keyboard('{ArrowLeft}')
    expect(dir).toHaveFocus()
  })

  it('keeps one row in the tab order', async () => {
    renderTree()
    const items = screen.getAllByRole('treeitem')
    expect(items.filter((i) => i.tabIndex === 0)).toHaveLength(0)
    expect(screen.getByRole('tree')).toHaveAttribute('tabindex', '0')

    await userEvent.click(items[1] as HTMLElement)
    expect(items.filter((i) => i.tabIndex === 0)).toEqual([items[1]])
  })
})

describe('ContentTree context menu', () => {
  it('offers the page actions for a file', async () => {
    const onRename = vi.fn<(path: string) => void>()
    const onDelete = vi.fn<(path: string) => void>()
    const onNewPage = vi.fn<(node: ContentNode) => void>()
    const onSelect = renderTree({ onRename, onDelete, onNewPage })
    const row = screen.getByText('a.md')

    await userEvent.pointer({ keys: '[MouseRight]', target: row })
    await userEvent.click(await screen.findByRole('menuitem', { name: 'Rename' }))
    expect(onRename).toHaveBeenCalledWith('content/posts/a.md')

    await userEvent.pointer({ keys: '[MouseRight]', target: row })
    await userEvent.click(await screen.findByRole('menuitem', { name: 'Delete' }))
    expect(onDelete).toHaveBeenCalledWith('content/posts/a.md')

    await userEvent.pointer({ keys: '[MouseRight]', target: row })
    await userEvent.click(await screen.findByRole('menuitem', { name: 'New page here' }))
    expect(onNewPage).toHaveBeenCalledWith(expect.objectContaining({ path: 'content/posts/a.md' }))

    await userEvent.pointer({ keys: '[MouseRight]', target: row })
    await userEvent.click(await screen.findByRole('menuitem', { name: 'Open' }))
    expect(onSelect).toHaveBeenCalledWith('content/posts/a.md')
  })

  it('offers only folder actions for a folder', async () => {
    renderTree({
      onRename: vi.fn<() => void>(),
      onDelete: vi.fn<() => void>(),
      onNewPage: vi.fn<() => void>(),
    })

    await userEvent.pointer({ keys: '[MouseRight]', target: screen.getByText('posts') })

    expect(await screen.findByRole('menuitem', { name: 'New page here' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'Copy path' })).toBeInTheDocument()
    expect(screen.queryByRole('menuitem', { name: 'Rename' })).not.toBeInTheDocument()
    expect(screen.queryByRole('menuitem', { name: 'Delete' })).not.toBeInTheDocument()
  })

  it('copies the path', async () => {
    const writeText = vi.fn<(text: string) => Promise<void>>(() => Promise.resolve())
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    renderTree()

    await userEvent.pointer({ keys: '[MouseRight]', target: screen.getByText('z.md') })
    await userEvent.click(await screen.findByRole('menuitem', { name: 'Copy path' }))

    expect(writeText).toHaveBeenCalledWith('content/z.md')
    vi.unstubAllGlobals()
  })
})
