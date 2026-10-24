import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

type Listener = () => void

function stubMedia(dark: boolean) {
  const listeners: Listener[] = []
  const mql = {
    matches: dark,
    addEventListener: (_: string, l: Listener) => listeners.push(l),
  }
  vi.stubGlobal('matchMedia', () => mql)
  return {
    flip(next: boolean) {
      mql.matches = next
      listeners.forEach((l) => l())
    },
  }
}

async function load() {
  vi.resetModules()
  return (await import('../theme')).useThemeStore
}

beforeEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove('dark')
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('theme store', () => {
  it('follows the OS and reacts to changes', async () => {
    const os = stubMedia(true)
    const store = await load()
    expect(store.getState().resolved).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)

    os.flip(false)
    expect(store.getState().resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('persists an override and ignores the OS while set', async () => {
    const os = stubMedia(false)
    const store = await load()
    store.getState().setChoice('dark')
    expect(localStorage.getItem('studio.theme')).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)

    os.flip(true)
    os.flip(false)
    expect(store.getState().resolved).toBe('dark')

    store.getState().setChoice('system')
    expect(localStorage.getItem('studio.theme')).toBeNull()
    expect(store.getState().resolved).toBe('light')
  })

  it('restores a stored override', async () => {
    stubMedia(false)
    localStorage.setItem('studio.theme', 'dark')
    const store = await load()
    expect(store.getState().choice).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })
})
