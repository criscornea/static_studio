export type SSGKind = 'hugo' | 'astro'

export interface Project {
  root: string
  kind: SSGKind
  configFile: string
  contentDir: string
  assetDir: string
  id: string
}

export interface ContentNode {
  name: string
  path: string
  isDir: boolean
  size?: number
  ext?: string
  // Present when the file has readable frontmatter.
  fields?: PageFields
  children?: ContentNode[]
}

export interface ContentTree {
  root: ContentNode
}

/** Stable error codes returned by the backend. */
export type ApiErrorCode =
  | 'bad_request'
  | 'missing_path'
  | 'not_a_project'
  | 'unsupported_generator'
  | 'cannot_open'
  | 'no_project'
  | 'stale_project'
  | 'cannot_read_content'
  | 'internal'
  | 'unknonw'
  | 'not_editable'
  | 'page_not_found'
  | 'page_too_large'
  | 'invalid_frontmatter'
  | 'cannot_read_page'
  | 'missing_query'
  | 'cannot_search'
  | 'invalid_name'
  | 'invalid_folder'
  | 'unknown_archetype'
  | 'unsupported_archetype'
  | 'invalid_archetype'
  | 'page_exists'
  | 'cannot_create_page'
  | 'cannot_rename_page'
  | 'cannot_delete_page'
  | 'cannot_save_page'
  | 'page_conflict'

/** An error response from the backend, or a transport failure. */
export class ApiError extends Error {
  readonly code: ApiErrorCode
  readonly status: number

  constructor(code: ApiErrorCode, message: string, status: number) {
    super(message)

    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

export type FrontmatterFormat = '' | 'yaml' | 'toml'

export interface PageFields {
  title: string
  // RFC 3339 string. JSON has no date type, so this is never a Date.
  date?: string
  draft: boolean
  tags?: string[]
}

export interface Page {
  path: string
  format: FrontmatterFormat
  fields: PageFields
  body: string
  /** sha256 of the file bytes; sent back on save. */
  hash: string
}

/** A template a new page can start from. */
export interface Archetype {
  name: string
  // True for the fallback used when the project defines no default.
  builtin?: boolean
}

export interface NewPageRequest {
  // Empty lets the backend pick: the archetype named after the folder, else the default.
  archetype: string
  // Relative to the content directory. Empty means its root.
  folder: string
  // What the user typed. The backend derives the file name from it.
  name: string
}

export interface RenamePageRequest {
  // Full paths of the page, starting with the content directory.
  from: string
  to: string
}

export interface SearchHit {
  path: string
  title: string
  draft: boolean
  // Text around the first body match. Absent when only the title or tags match.
  snippet?: string
}

export interface SearchResults {
  hits: SearchHit[]
  // True when more pages matched than the backend returns.
  truncated: boolean
}

/** Normalises anything thrown by the client into an Api Error. */
export function asApiError(err: unknown): ApiError {
  if (err instanceof ApiError) {
    return err
  }
  return new ApiError('unknonw', 'Something went wrong. Please try again.', 0)
}
