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
  | 'stable_project'
  | 'cannot_read_content'
  | 'internal'
  | 'unknonw'
  | 'not_editable'
  | 'page_not_found'
  | 'page_too_large'
  | 'invalid_frontmatter'
  | 'cannot_read_page'

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
}

/** Normalises anything thrown by the client into an Api Error. */
export function asApiError(err: unknown): ApiError {
  if (err instanceof ApiError) {
    return err
  }
  return new ApiError('unknonw', 'Something went wrong. Please try again.', 0)
}
