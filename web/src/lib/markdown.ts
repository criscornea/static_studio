/** Markdown image link for an uploaded file. */
export function imageMarkdown(path: string, file: string): string {
  const alt = file.replace(/\.[^.]*$/, '').replace(/[[\]]/g, '')
  return `![${alt}](${path})`
}
