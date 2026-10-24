import { create } from 'zustand'

export type ThemeChoice = 'system' | 'light' | 'dark'

export const themeKey = 'studio.theme'

interface ThemeState {
  choice: ThemeChoice
  /** What is actually applied after resolving "system". */
  resolved: 'light' | 'dark'
  setChoice: (choice: ThemeChoice) => void
}

const query = '(prefers-color-scheme: dark)'

function media(): MediaQueryList | null {
  return typeof window !== 'undefined' && typeof window.matchMedia === 'function'
    ? window.matchMedia(query)
    : null
}

function readChoice(): ThemeChoice {
  try {
    const v = localStorage.getItem(themeKey)
    return v === 'light' || v === 'dark' ? v : 'system'
  } catch {
    return 'system'
  }
}

function resolve(choice: ThemeChoice): 'light' | 'dark' {
  if (choice !== 'system') {
    return choice
  }
  return media()?.matches ? 'dark' : 'light'
}

function apply(resolved: 'light' | 'dark') {
  document.documentElement.classList.toggle('dark', resolved === 'dark')
  document.documentElement.style.colorScheme = resolved
}

const initial = readChoice()

/** Light or dark theme, following the OS unless the user overrides it. */
export const useThemeStore = create<ThemeState>()((set) => ({
  choice: initial,
  resolved: resolve(initial),
  setChoice(choice) {
    try {
      if (choice === 'system') {
        localStorage.removeItem(themeKey)
      } else {
        localStorage.setItem(themeKey, choice)
      }
    } catch {
      // Storage may be blocked; the choice then lasts for this session only.
    }
    set({ choice, resolved: resolve(choice) })
  },
}))

apply(useThemeStore.getState().resolved)
useThemeStore.subscribe((s) => apply(s.resolved))

media()?.addEventListener('change', () => {
  const { choice } = useThemeStore.getState()
  if (choice === 'system') {
    useThemeStore.setState({ resolved: resolve('system') })
  }
})
