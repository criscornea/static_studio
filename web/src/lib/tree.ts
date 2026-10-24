import type { ContentNode } from '@/api/types'

export type SortKey = 'name' | 'title' | 'date-desc' | 'date-asc'
export type StatusFilter = 'all' | 'draft' | 'published'

/** How the content tree is shown. The default shows everything in name order. */
export interface TreeView {
  sort: SortKey
  status: StatusFilter
  // null shows every tag.
  tag: string | null
}

export const defaultView: TreeView = { sort: 'name', status: 'all', tag: null }

export function isDefaultView(view: TreeView): boolean {
  return view.sort === defaultView.sort && view.status === defaultView.status && view.tag === null
}

/** Every distinct tag under the nodes, alphabetically. */
export function collectTags(nodes: ContentNode[]): string[] {
  const tags = new Set<string>()
  const visit = (node: ContentNode) => {
    node.fields?.tags?.forEach((tag) => tags.add(tag))
    node.children?.forEach(visit)
  }
  nodes.forEach(visit)
  return [...tags].sort((a, b) => a.localeCompare(b, undefined, { sensitivity: 'base' }))
}

/**
 * Filters and sorts the nodes for display. Folders stay first, ordered by
 * name, and folders left without a matching page are dropped. A page without
 * frontmatter counts as published and has no tags.
 */
export function applyView(nodes: ContentNode[], view: TreeView): ContentNode[] {
  const out: ContentNode[] = []

  for (const node of nodes) {
    if (node.isDir) {
      const children = applyView(node.children ?? [], view)
      // A folder is only worth showing when something in it matches.
      if (children.length > 0 || isDefaultView(view)) {
        out.push({ ...node, children })
      }
    } else if (matches(node, view)) {
      out.push(node)
    }
  }

  return out.sort(compareFor(view.sort))
}

function matches(node: ContentNode, view: TreeView): boolean {
  const draft = node.fields?.draft ?? false
  if (view.status === 'draft' && !draft) {
    return false
  }
  if (view.status === 'published' && draft) {
    return false
  }
  return view.tag === null || (node.fields?.tags?.includes(view.tag) ?? false)
}

const collator = new Intl.Collator(undefined, { sensitivity: 'base', numeric: true })

function compareFor(sort: SortKey): (a: ContentNode, b: ContentNode) => number {
  return (a, b) => {
    if (a.isDir !== b.isDir) {
      return a.isDir ? -1 : 1
    }
    if (a.isDir || sort === 'name') {
      return collator.compare(a.name, b.name)
    }
    if (sort === 'title') {
      return collator.compare(a.fields?.title || a.name, b.fields?.title || b.name)
    }
    return compareDates(a, b, sort === 'date-desc') || collator.compare(a.name, b.name)
  }
}

/** Orders by date, with pages that have no usable date always last. */
function compareDates(a: ContentNode, b: ContentNode, newestFirst: boolean): number {
  const da = timeOf(a)
  const db = timeOf(b)
  if (da === null || db === null) {
    return da === db ? 0 : da === null ? 1 : -1
  }
  return newestFirst ? db - da : da - db
}

function timeOf(node: ContentNode): number | null {
  const raw = node.fields?.date
  const time = raw ? Date.parse(raw) : NaN
  return Number.isNaN(time) ? null : time
}

/** Every folder under the content directory, as paths relative to it. */
export function collectFolders(root: ContentNode | null): string[] {
  const folders: string[] = []
  const visit = (node: ContentNode, prefix: string) => {
    for (const child of node.children ?? []) {
      if (child.isDir) {
        const folder = prefix + child.name
        folders.push(folder)
        visit(child, folder + '/')
      }
    }
  }
  if (root) {
    visit(root, '')
  }
  return folders.sort((a, b) => a.localeCompare(b))
}

/** The folder of a page path, relative to the content directory. Empty for the root. */
export function folderOf(pagePath: string | null, contentDir: string): string {
  if (!pagePath?.startsWith(contentDir + '/')) {
    return ''
  }
  const relative = pagePath.slice(contentDir.length + 1)
  const slash = relative.lastIndexOf('/')
  return slash < 0 ? '' : relative.slice(0, slash)
}

export interface PageEntry {
  path: string
  title: string
  draft: boolean
}

/** Every file in the tree as a flat list, in tree order. */
export function flattenPages(nodes: ContentNode[]): PageEntry[] {
  return nodes.flatMap((n) =>
    n.isDir
      ? flattenPages(n.children ?? [])
      : [{ path: n.path, title: n.fields?.title || n.name, draft: n.fields?.draft === true }],
  )
}
