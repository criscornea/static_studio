/**
 * Shortcut strings look like "Mod+Shift+P". Mod is Ctrl, or Cmd on macOS; either
 * is accepted on every platform so the bindings work the same everywhere.
 */
interface Parsed {
  mod: boolean
  shift: boolean
  alt: boolean
  key: string
}

export function parseShortcut(shortcut: string): Parsed {
  const parts = shortcut.split('+')
  const key = (parts.pop() ?? '').toLowerCase()
  const mods = parts.map((p) => p.toLowerCase())
  return {
    mod: mods.includes('mod'),
    shift: mods.includes('shift'),
    alt: mods.includes('alt'),
    key,
  }
}

/** True when the shortcut needs Ctrl or Cmd, which makes it safe inside text fields. */
export function usesMod(shortcut: string): boolean {
  return parseShortcut(shortcut).mod
}

export function matchesShortcut(e: KeyboardEvent, shortcut: string): boolean {
  const p = parseShortcut(shortcut)
  if (p.mod !== (e.ctrlKey || e.metaKey) || p.shift !== e.shiftKey || p.alt !== e.altKey) {
    return false
  }
  const key = e.key.toLowerCase()
  if (key === p.key) {
    return true
  }
  // Alt can change the typed character on macOS; the physical key is stable.
  return p.key.length === 1 && /[a-z]/.test(p.key) && e.code === `Key${p.key.toUpperCase()}`
}

function isMac(): boolean {
  return typeof navigator !== 'undefined' && /mac|iphone|ipad/i.test(navigator.platform)
}

/** Human-readable form, e.g. "Ctrl+Shift+P" or "⌘⇧P". */
export function formatShortcut(shortcut: string, mac = isMac()): string {
  const p = parseShortcut(shortcut)
  const key =
    p.key.length === 1 ? p.key.toUpperCase() : p.key.charAt(0).toUpperCase() + p.key.slice(1)
  if (mac) {
    return `${p.mod ? '⌘' : ''}${p.alt ? '⌥' : ''}${p.shift ? '⇧' : ''}${key}`
  }
  return [p.mod && 'Ctrl', p.alt && 'Alt', p.shift && 'Shift', key].filter(Boolean).join('+')
}
