import type { ApiErrorCode } from '@/api/types'

/**
 * Readable messages for error codes whose server text is generic or
 * technical. Codes not listed keep the server's own message, which already
 * carries the specifics (for example which name was invalid).
 */
const messages: Partial<Record<ApiErrorCode, string>> = {
  internal: 'Something went wrong inside the editor. Please try again.',
  no_project: 'No project is open. Open a project first.',
  stale_project: 'The project was changed or reopened. Reload it and try again.',
  cannot_open: 'The project could not be opened. Check that the folder exists.',
  cannot_read_content: 'The content folder could not be read.',
  cannot_read_page: 'The page could not be read.',
  cannot_save_page: 'The page could not be saved. Your changes are still in the editor.',
  cannot_create_page: 'The page could not be created.',
  cannot_rename_page: 'The page could not be renamed.',
  cannot_delete_page: 'The page could not be deleted.',
  cannot_save_asset: 'The image could not be saved.',
  cannot_search: 'The search failed. Please try again.',
  page_conflict: 'The page changed on disk since you opened it. Reload it before saving.',
  page_not_found: 'That page no longer exists.',
  page_too_large: 'That page is too large to edit here.',
  build_running: 'A build is already running.',
  devserver_running: 'The preview server is already running.',
  devserver_failed: 'The preview server could not be started.',
  publish_running: 'A publish is already running.',
  recent_failed: 'The list of recent projects could not be read.',
}

/** Returns a readable message for the code, or the server's own text. */
export function readableMessage(code: string, serverMessage: string): string {
  return messages[code as ApiErrorCode] ?? serverMessage
}
