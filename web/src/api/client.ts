import { ApiError, type ApiErrorCode, type ContentTree, type Project } from "./types";

/**
 * The id of the open project, sent with every request that touches project
 * files. The backend rejects a stale id, so a reopened project cannot be
 * written to by a stale view.
 */
let projectID: string | null = null

export function setProjectID(id: string | null): void {
  projectID = id
}

interface RequestOptions {
  method?: 'GET' | 'POST'
  body?: unknown
  // Send the project id. Defaults to true.
  withProject?: boolean
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, withProject = true } = options

  const headers = new Headers()
  if (body != undefined) {
    headers.set('Content-Type', 'application/json')
  }
  if (withProject && projectID !== null) {
    headers.set('X-Project-ID', projectID)
  }

  let res: Response
  try {
    res = await fetch(path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError('internal', 'The editor could not reach its backend. Is static_studio still running?', 0)
  }

  if (res.status === 204) {
    return undefined as T
  }

  const payload: unknown = await res.json().catch(() => null)

  if (!res.ok) {
    throw toApiError(payload, res.status)
  }
  return payload as T
}

function toApiError(payload: unknown, status: number): ApiError {
  if (
    typeof payload === 'object' &&
    payload !== null &&
    'code' in payload &&
    'message' in payload &&
    typeof payload.code === 'string' &&
    typeof payload.message === 'string'
  ) {
    return new ApiError(payload.code as ApiErrorCode, payload.message, status)
  }

  return new ApiError('unknonw', `The backend returned an unexpected response (${status})`, status)
}

export const api = {
  health: () => request<{ status: string }>('/api/health', { withProject: false }),
  openProject: (path: string) =>
    request<Project>('/api/project/open', {
      method: 'POST',
      body: { path },
      withProject: false,
    }),
  currentProject: () => request<Project>('/api/project', { withProject: false }),
  closeProject: () => request<void>('/api/project/close', { method: 'POST', withProject: false }),
  contentTree: () => request<ContentTree>('/api/content'),
}
