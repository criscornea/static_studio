import { beforeEach, describe, expect, it, vi } from 'vitest'

async function load() {
  vi.resetModules()
  return (await import('../editorMode')).useEditorModeStore
}

beforeEach(() => localStorage.clear())

describe('editor mode store', () => {
  it('defaults to live preview', async () => {
    expect((await load()).getState().mode).toBe('live')
  })

  it('persists the choice across loads', async () => {
    const store = await load()
    store.getState().toggle()
    expect(store.getState().mode).toBe('source')
    expect(localStorage.getItem('studio.editorMode')).toBe('source')
    expect((await load()).getState().mode).toBe('source')
    ;(await load()).getState().setMode('live')
    expect((await load()).getState().mode).toBe('live')
  })

  it('ignores junk in storage', async () => {
    localStorage.setItem('studio.editorMode', 'nonsense')
    expect((await load()).getState().mode).toBe('live')
  })
})
