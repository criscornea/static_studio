export type SSGKind = 'hugo' | 'astro'

export interface Project {
  root: string
  kind: SSGKind
  configFile: string
  contentDir: string
  // Every content dir, primary first. Mirrors ssg.ContentDirs.
  contentDirs: ContentDirInfo[]
  assetDir: string
  id: string
}

export interface ContentDirInfo {
  path: string
  // Language the dir belongs to, when it is language-specific.
  lang?: string
}

export interface RecentProject {
  path: string
  name: string
  kind: SSGKind
  openedAt: string
  // False when the folder no longer exists.
  available: boolean
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

/** The tree of one content dir. */
export interface ContentRoot {
  dir: string
  lang?: string
  root: ContentNode
}

export interface ContentTree {
  // The primary tree, repeated from roots[0].
  root: ContentNode
  roots: ContentRoot[]
}

/** Stable error codes returned by the backend. */
export type ApiErrorCode =
  | 'bad_request'
  | 'missing_path'
  | 'not_a_project'
  | 'unsupported_generator'
  | 'invalid_config'
  | 'cannot_open'
  | 'no_project'
  | 'stale_project'
  | 'cannot_read_content'
  | 'devserver_running'
  | 'devserver_failed'
  | 'publish_running'
  | 'invalid_publish_settings'
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
  | 'asset_too_large'
  | 'asset_type'
  | 'asset_name'
  | 'asset_exists'
  | 'cannot_save_asset'
  | 'build_running'
  | 'recent_failed'

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
  /** 1-based line of the file where the body starts. */
  bodyLine: number
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
  // Content dir to create the page in. Omitted means the primary one.
  dir?: string
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

/** Where the site is published. */
export interface PublishSettings {
  target: 'github-pages' | 'rsync'
  remote: string
  branch: string
  rsyncDest: string
  rsyncPort: number
  rsyncIdentity: string
  rsyncDelete: boolean
  rsyncDryRun: boolean
}

/** One server-sent event from the build stream. */
export interface BuildEvent {
  kind: 'line' | 'done'
  line?: string
  ok?: boolean
  error?: string
  problems?: BuildProblem[]
}

/** A build error tied to a line of a project file. */
export interface BuildProblem {
  /** Slash-separated, relative to the project root. */
  file: string
  line: number
  column?: number
  message: string
}

/** State of the Hugo preview server. */
export interface DevServerStatus {
  running: boolean
  // True once the server accepts connections.
  ready: boolean
  // Origin of the server, for example http://127.0.0.1:41234.
  url?: string
  // Full URL of the requested page. Only set when a path was asked for.
  pageUrl?: string
  // Last output lines of a server that exited on its own.
  output?: string[]
}
