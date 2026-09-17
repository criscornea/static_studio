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
