/** Renders text with every case-insensitive occurrence of query marked. */
export function Highlight({ text, query }: { text: string; query: string }) {
  const needle = query.trim()
  if (needle === '') {
    return text
  }

  // The capture group makes split keep the matches, at the odd indexes.
  const parts = text.split(new RegExp(`(${escapeRegExp(needle)})`, 'i'))
  return parts.map((part, i) =>
    i % 2 === 1 ? (
      <mark key={i} className="rounded-sm bg-yellow-200 px-0.5 text-inherit dark:bg-yellow-500/40">
        {part}
      </mark>
    ) : (
      part
    ),
  )
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
