import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ContentNode } from '@/api/types'
import { defaultView } from '@/lib/tree'
import { useProjectStore } from '@/stores/project'
import { useSearchStore } from '@/stores/search'
import { useViewStore } from '@/stores/view'
import { useUiStore } from '@/stores/ui'
import { AppDialogs } from '../AppDialogs'
import { Sidebar } from '../Sidebar'

const tree: ContentNode = {
  name: 'content',
  path: 'content',
  isDir: true,
  children: [
    {
      name: 'live.md',
      path: 'content/live.md',
      isDir: false,
      fields: { title: 'Live Post', draft: false },
    },
    {
      name: 'wip.md',
      path: 'content/wip.md',
      isDir: false,
      fields: { title: 'Work In Progress', draft: true },
    },
  ],
}

beforeEach(() => {
  useProjectStore.setState({ tree })
  useViewStore.setState(defaultView)
  useSearchStore.getState().clear()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('Sidebar', () => {
  it('lists every page by default', () => {
    render(<Sidebar />)

    expect(screen.getByText('Live Post')).toBeInTheDocument()
    expect(screen.getByText('Work In Progress')).toBeInTheDocument()
  })

  it('shows only drafts when filtered, and offers a way back when nothing matches', async () => {
    render(<Sidebar />)

    useViewStore.getState().setStatus('draft')
    expect(await screen.findByText('Work In Progress')).toBeInTheDocument()
    expect(screen.queryByText('Live Post')).not.toBeInTheDocument()

    useViewStore.getState().setTag('missing')
    expect(await screen.findByText('No pages match this filter.')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Show all pages' }))
    expect(await screen.findByText('Live Post')).toBeInTheDocument()
  })

  it('has no search box', () => {
    render(<Sidebar />)

    expect(screen.queryByRole('searchbox')).not.toBeInTheDocument()
  })
})

describe('Sidebar new page', () => {
  it('opens the new page dialog from the plus button', async () => {
    vi.stubGlobal('fetch', () =>
      Promise.resolve(new Response(JSON.stringify({ archetypes: [] }), { status: 200 })),
    )
    render(
      <>
        <Sidebar />
        <AppDialogs />
      </>,
    )

    await userEvent.click(screen.getByRole('button', { name: 'New page' }))

    expect(await screen.findByLabelText('Name')).toBeInTheDocument()
  })
})

describe('Sidebar empty and loading states', () => {
  it('shows skeletons while the tree loads', () => {
    useProjectStore.setState({ tree: null })
    render(<Sidebar />)
    expect(screen.getByRole('status')).toHaveTextContent('Loading pages')
  })

  it('offers a new page when the content dir is empty', async () => {
    useProjectStore.setState({
      tree: { name: 'content', path: 'content', isDir: true, children: [] },
    })
    render(<Sidebar />)
    expect(screen.getByText('No content files yet.')).toBeInTheDocument()
    await userEvent.click(screen.getAllByRole('button', { name: 'New page' }).at(-1)!)
    expect(useUiStore.getState().newPage).not.toBeNull()
  })
})

describe('Sidebar with several content dirs', () => {
  afterEach(() => useProjectStore.setState({ roots: [] }))

  it('renders one labelled tree per content dir', () => {
    const page = (path: string, title: string): ContentNode => ({
      name: path.split('/').pop()!,
      path,
      isDir: false,
      fields: { title, draft: false },
    })
    const dir = (path: string, child: ContentNode): ContentNode => ({
      name: path,
      path,
      isDir: true,
      children: [child],
    })
    const en = dir('content', page('content/a.md', 'English page'))
    const de = dir('inhalt', page('inhalt/b.md', 'German page'))
    useProjectStore.setState({
      tree: en,
      roots: [
        { dir: 'content', root: en },
        { dir: 'inhalt', lang: 'de', root: de },
      ],
    })
    render(<Sidebar />)

    expect(screen.getAllByTestId('content-root')).toHaveLength(2)
    expect(screen.getByRole('tree', { name: 'Pages in inhalt' })).toBeInTheDocument()
    expect(screen.getByText('de')).toBeInTheDocument()
    expect(screen.getByText('English page')).toBeInTheDocument()
    expect(screen.getByText('German page')).toBeInTheDocument()
  })
})

describe('openNewPage', () => {
  it('carries the content dir into the dialog state', () => {
    useUiStore.getState().openNewPage('sub', 'inhalt')
    expect(useUiStore.getState().newPage).toEqual({ folder: 'sub', dir: 'inhalt' })
    useUiStore.getState().closeNewPage()
  })
})
