import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { buildCommands, dispatchShortcut } from '@/lib/commands'
import { useEditorStore } from '@/stores/editor'
import { usePageStore } from '@/stores/page'
import { MarkdownEditor } from '../MarkdownEditor'

afterEach(() => {
  useEditorStore.setState({ view: null, focused: false, active: [] })
  usePageStore.setState({ page: null })
})

function setup(value = 'hello') {
  const onChange = vi.fn<(text: string) => void>()
  const utils = render(<MarkdownEditor value={value} onChange={onChange} onSave={() => {}} />)
  const content = utils.container.querySelector<HTMLElement>('.cm-content')!
  return { ...utils, content, onChange, view: useEditorStore.getState }
}

describe('FormatToolbar visibility', () => {
  it('shows only while the editor has focus', () => {
    const { content } = setup()
    expect(screen.queryByRole('toolbar')).toBeNull()
    act(() => content.dispatchEvent(new FocusEvent('focus')))
    expect(screen.getByRole('toolbar')).toBeTruthy()
    act(() => content.dispatchEvent(new FocusEvent('blur')))
    expect(screen.queryByRole('toolbar')).toBeNull()
  })

  it('stays when focus moves to the toolbar', () => {
    const { content } = setup()
    act(() => content.dispatchEvent(new FocusEvent('focus')))
    const button = screen.getByRole('button', { name: 'Bold' })
    act(() => content.dispatchEvent(new FocusEvent('blur', { relatedTarget: button })))
    expect(screen.getByRole('toolbar')).toBeTruthy()
  })

  it('is gone with the editor', () => {
    const { content, unmount } = setup()
    act(() => content.dispatchEvent(new FocusEvent('focus')))
    unmount()
    expect(screen.queryByRole('toolbar')).toBeNull()
    expect(useEditorStore.getState().view).toBeNull()
  })

  it('keeps editor focus on mousedown and formats on click', () => {
    const { content, onChange } = setup('hello')
    act(() => content.dispatchEvent(new FocusEvent('focus')))
    const view = useEditorStore.getState().view!
    view.dispatch({ selection: { anchor: 0, head: 5 } })
    const bold = screen.getByRole('button', { name: 'Bold' })
    expect(fireEvent.mouseDown(bold)).toBe(false) // default prevented
    fireEvent.click(bold)
    expect(onChange).toHaveBeenLastCalledWith('**hello**')
  })

  it('highlights the active formats', () => {
    const { content } = setup('**bo**')
    act(() => content.dispatchEvent(new FocusEvent('focus')))
    const view = useEditorStore.getState().view!
    act(() => view.dispatch({ selection: { anchor: 3 } }))
    expect(screen.getByRole('button', { name: 'Bold' }).getAttribute('aria-pressed')).toBe('true')
    expect(screen.getByRole('button', { name: 'Italic' }).getAttribute('aria-pressed')).toBe(
      'false',
    )
  })
})

describe('editor shortcuts', () => {
  function key(target: Element, k: string) {
    return fireEvent.keyDown(target, { key: k, ctrlKey: true })
  }

  it('format inside the editor and leave the global handlers alone', () => {
    const { content, onChange } = setup('hello')
    const sidebar = vi.fn<() => void>()
    const search = vi.fn<() => void>()
    const commands = [
      { id: 'view.sidebar', shortcut: 'Mod+B', run: sidebar, enabled: true },
      { id: 'search.focus', shortcut: 'Mod+K', run: search, enabled: true },
    ] as never[]
    const listener = (e: KeyboardEvent) => dispatchShortcut(e, commands)
    window.addEventListener('keydown', listener)

    act(() => content.dispatchEvent(new FocusEvent('focus')))
    useEditorStore.getState().view!.dispatch({ selection: { anchor: 0, head: 5 } })
    key(content, 'b')
    expect(onChange).toHaveBeenLastCalledWith('**hello**')
    key(content, 'k')
    expect(onChange.mock.calls.at(-1)?.[0]).toContain('](url)')
    expect(sidebar).not.toHaveBeenCalled()
    expect(search).not.toHaveBeenCalled()

    // Outside the editor the global shortcuts still work.
    key(document.body, 'b')
    key(document.body, 'k')
    expect(sidebar).toHaveBeenCalledTimes(1)
    expect(search).toHaveBeenCalledTimes(1)
    window.removeEventListener('keydown', listener)
  })

  it('are ignored by the global handler even for the registry commands', () => {
    const commands = buildCommands()
    const bold = commands.find((c) => c.id === 'format.bold')!
    expect(bold.scope).toBe('editor')
    const run = vi.spyOn(bold, 'run')
    const e = new KeyboardEvent('keydown', { key: 'b', ctrlKey: true, cancelable: true })
    dispatchShortcut(e, [bold])
    expect(run).not.toHaveBeenCalled()
  })
})

describe('format commands in the registry', () => {
  it('are disabled without an editor and run on the last focused one', () => {
    expect(buildCommands().find((c) => c.id === 'format.bold')?.enabled).toBe(false)

    const { content, onChange } = setup('hello')
    usePageStore.setState({ page: { path: 'a.md' } as never })
    act(() => content.dispatchEvent(new FocusEvent('focus')))
    useEditorStore.getState().view!.dispatch({ selection: { anchor: 0, head: 5 } })
    act(() => content.dispatchEvent(new FocusEvent('blur')))

    const h2 = buildCommands().find((c) => c.id === 'format.h2')!
    expect(h2.group).toBe('Format')
    expect(h2.enabled).toBe(true)
    h2.run()
    expect(onChange).toHaveBeenLastCalledWith('## hello')
  })
})
