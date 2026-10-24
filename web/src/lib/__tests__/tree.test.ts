import { describe, expect, it } from 'vitest'

import type { ContentNode, PageFields } from '@/api/types'
import {
  applyView,
  collectFolders,
  collectTags,
  defaultView,
  flattenPages,
  folderOf,
  type TreeView,
} from '../tree'

const page = (name: string, fields?: Partial<PageFields>): ContentNode => ({
  name,
  path: `content/${name}`,
  isDir: false,
  fields: fields && { title: '', draft: false, ...fields },
})

const dir = (name: string, ...children: ContentNode[]): ContentNode => ({
  name,
  path: `content/${name}`,
  isDir: true,
  children,
})

const view = (v: Partial<TreeView>): TreeView => ({ ...defaultView, ...v })
const names = (nodes: ContentNode[]) => nodes.map((n) => n.name)

const nodes = [
  dir('posts', page('old.md', { title: 'Old', date: '2020-01-01T00:00:00Z', tags: ['go'] })),
  dir('empty'),
  page('b.md', { title: 'Zebra', date: '2024-05-01T00:00:00Z', draft: true, tags: ['go', 'web'] }),
  page('a.md', { title: 'apple', date: '2023-01-01T00:00:00Z' }),
  page('c.md', { title: 'Mango' }),
  page('plain.md'),
]

describe('applyView default', () => {
  it('keeps every node, folders first, in the given name order', () => {
    const out = applyView(nodes, defaultView)

    expect(names(out)).toEqual(['empty', 'posts', 'a.md', 'b.md', 'c.md', 'plain.md'])
  })

  it('does not change its input', () => {
    const before = JSON.stringify(nodes)
    applyView(nodes, view({ sort: 'title', status: 'draft' }))

    expect(JSON.stringify(nodes)).toBe(before)
  })
})

describe('applyView sort', () => {
  const files = (sort: TreeView['sort']) =>
    names(applyView(nodes, view({ sort })).filter((n) => !n.isDir))

  it('sorts by title, ignoring case, falling back to the filename', () => {
    expect(files('title')).toEqual(['a.md', 'c.md', 'plain.md', 'b.md'])
  })

  it('sorts newest first with undated pages last', () => {
    expect(files('date-desc')).toEqual(['b.md', 'a.md', 'c.md', 'plain.md'])
  })

  it('sorts oldest first with undated pages still last', () => {
    expect(files('date-asc')).toEqual(['a.md', 'b.md', 'c.md', 'plain.md'])
  })

  it('treats an unparseable date as no date', () => {
    const out = applyView(
      [page('bad.md', { date: 'next tuesday' }), page('good.md', { date: '2020-01-01T00:00:00Z' })],
      view({ sort: 'date-desc' }),
    )

    expect(names(out)).toEqual(['good.md', 'bad.md'])
  })

  it('sorts inside folders too', () => {
    const out = applyView(
      [dir('d', page('x.md', { title: 'B' }), page('y.md', { title: 'A' }))],
      view({ sort: 'title' }),
    )

    expect(names(out[0]?.children ?? [])).toEqual(['y.md', 'x.md'])
  })
})

describe('applyView filter', () => {
  it('shows only drafts and drops folders without any', () => {
    const out = applyView(nodes, view({ status: 'draft' }))

    expect(names(out)).toEqual(['b.md'])
  })

  it('shows only published pages, counting a page without frontmatter as published', () => {
    const out = applyView(nodes, view({ status: 'published' }))

    expect(names(out)).toEqual(['posts', 'a.md', 'c.md', 'plain.md'])
  })

  it('filters by tag, keeping the folders that hold a match', () => {
    const out = applyView(nodes, view({ tag: 'go' }))

    expect(names(out)).toEqual(['posts', 'b.md'])
    expect(names(out[0]?.children ?? [])).toEqual(['old.md'])
  })

  it('combines status and tag', () => {
    const out = applyView(nodes, view({ status: 'published', tag: 'go' }))

    expect(names(out)).toEqual(['posts'])
  })

  it('shows nothing when no page matches', () => {
    expect(applyView(nodes, view({ tag: 'missing' }))).toEqual([])
  })
})

describe('collectTags', () => {
  it('finds distinct tags at any depth, alphabetically', () => {
    expect(collectTags(nodes)).toEqual(['go', 'web'])
  })

  it('is empty when no page has tags', () => {
    expect(collectTags([page('a.md', { title: 'A' })])).toEqual([])
  })
})

describe('collectFolders', () => {
  it('lists nested folders relative to the content directory', () => {
    const root: ContentNode = {
      name: 'content',
      path: 'content',
      isDir: true,
      children: [
        {
          ...dir('posts', dir('2024')),
          children: [{ ...dir('2024'), path: 'content/posts/2024' }],
        },
        page('a.md'),
        dir('about'),
      ],
    }

    expect(collectFolders(root)).toEqual(['about', 'posts', 'posts/2024'])
  })

  it('is empty without a tree', () => {
    expect(collectFolders(null)).toEqual([])
  })
})

describe('folderOf', () => {
  it('gives the page folder relative to the content directory', () => {
    expect(folderOf('content/posts/2024/a.md', 'content')).toBe('posts/2024')
    expect(folderOf('content/posts/a.md', 'content')).toBe('posts')
  })

  it('is empty for a page at the top level, no page or a foreign path', () => {
    expect(folderOf('content/a.md', 'content')).toBe('')
    expect(folderOf(null, 'content')).toBe('')
    expect(folderOf('static/a.md', 'content')).toBe('')
    expect(folderOf('contents/a.md', 'content')).toBe('')
  })
})

describe('flattenPages', () => {
  it('lists files depth first with a title fallback', () => {
    const nodes: ContentNode[] = [
      {
        name: 'posts',
        path: 'content/posts',
        isDir: true,
        children: [
          {
            name: 'a.md',
            path: 'content/posts/a.md',
            isDir: false,
            fields: { title: 'A', draft: true },
          },
        ],
      },
      { name: 'b.md', path: 'content/b.md', isDir: false },
    ]
    expect(flattenPages(nodes)).toEqual([
      { path: 'content/posts/a.md', title: 'A', draft: true },
      { path: 'content/b.md', title: 'b.md', draft: false },
    ])
  })
})
