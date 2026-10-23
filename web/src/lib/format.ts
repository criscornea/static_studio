/**
 * Formats an RFC 3339 string for display. An unparseable value is returned
 * as written, so a hand-typed date still shows up. Empty input gives null.
 */
export function formatDate(
  raw: string | undefined,
  options: Intl.DateTimeFormatOptions = { dateStyle: 'medium', timeStyle: 'short' },
): string | null {
  if (!raw) {
    return null
  }
  const parsed = new Date(raw)
  return Number.isNaN(parsed.getTime())
    ? raw
    : new Intl.DateTimeFormat(undefined, options).format(parsed)
}
