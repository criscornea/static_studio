import { fireEvent } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { renderHook } from '@testing-library/react'
import { useCommandShortcuts } from '@/hooks/useCommandShortcuts'
import { useLayoutStore } from '@/stores/layout'
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { buildCommands, dispatchShortcut, filterCommands, type Command } from '../commands'
import { parseShortcut } from '../shortcuts'

beforeEach(() => {
  localStorage.clear()
  useLayoutStore.setState({ sidebarCollapsed: false, buildLogOpen: false })
  useUiStore.setState({ paletteOpen: false, quickOpenOpen: false, newPage: null })
  useProjectStore.setState({
    project: { id: 'p', root: '/x', kind: 'hugo', configFile: 'hugo.toml', contentDir: 'content' },
  } as never)
})

afterEach(() => {
  document.body.innerHTML = ''
  useProjectStore.setState({ project: null })
})

describe('registry', () => {
  it('lists every action with unique ids and unique shortcuts', () => {
    const commands = buildCommands()
    const ids = commands.map((c) => c.id)
    expect(new Set(ids).size).toBe(ids.length)
    for (const id of [
      'page.open',
      'page.new',
      'page.rename',
      'page.delete',
      'page.save',
      'view.sidebar',
      'view.preview',
      'view.buildlog',
      'site.build',
      'site.publish',
      'server.toggle',
      'theme.dark',
      'project.close',
      'palette',
    ]) {
      expect(ids).toContain(id)
    }
    const shortcuts = commands.flatMap((c) => (c.shortcut ? [c.shortcut.toLowerCase()] : []))
    expect(new Set(shortcuts).size).toBe(shortcuts.length)
    shortcuts.forEach((s) => expect(parseShortcut(s).key).not.toBe(''))
  })

  it('disables page commands without an open page', () => {
    const byId = Object.fromEntries(buildCommands().map((c) => [c.id, c]))
    expect(byId['page.rename']?.enabled).toBe(false)
    expect(byId['page.new']?.enabled).toBe(true)
  })
})

describe('filterCommands', () => {
  const list = [
    { id: 'a', title: 'Toggle sidebar', group: 'View' },
    { id: 'b', title: 'Toggle build log', group: 'View' },
    { id: 'c', title: 'Build site', group: 'Site' },
  ] as Command[]

  it('returns everything for an empty query', () => {
    expect(filterCommands(list, '  ')).toHaveLength(3)
  })

  it('requires every word, in any order, ignoring case', () => {
    expect(filterCommands(list, 'BUILD toggle').map((c) => c.id)).toEqual(['b'])
    expect(filterCommands(list, 'site').map((c) => c.id)).toEqual(['c'])
  })

  it('matches the group name', () => {
    expect(filterCommands(list, 'view').map((c) => c.id)).toEqual(['a', 'b'])
  })
})

describe('shortcut dispatch', () => {
  it('runs the matching command', () => {
    renderHook(() => useCommandShortcuts())
    fireEvent.keyDown(window, { key: 'b', ctrlKey: true })
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(true)
    fireEvent.keyDown(window, { key: 'j', metaKey: true })
    expect(useLayoutStore.getState().buildLogOpen).toBe(true)
  })

  it('opens the palette and quick open', () => {
    renderHook(() => useCommandShortcuts())
    fireEvent.keyDown(window, { key: 'P', ctrlKey: true, shiftKey: true })
    expect(useUiStore.getState().paletteOpen).toBe(true)
    fireEvent.keyDown(window, { key: 'p', ctrlKey: true })
    expect(useUiStore.getState().quickOpenOpen).toBe(true)
    expect(useUiStore.getState().paletteOpen).toBe(false)
  })

  it('ignores a plain letter and does nothing without a project', () => {
    renderHook(() => useCommandShortcuts())
    fireEvent.keyDown(window, { key: 'b' })
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(false)
    useProjectStore.setState({ project: null })
    fireEvent.keyDown(window, { key: 'b', ctrlKey: true })
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(false)
  })

  it('leaves a key the editor already handled', () => {
    const e = new KeyboardEvent('keydown', { key: 'b', ctrlKey: true, cancelable: true })
    e.preventDefault()
    expect(dispatchShortcut(e, buildCommands())).toBe(false)
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(false)
  })

  it('does not fire while a dialog is open', () => {
    document.body.innerHTML = '<div role="dialog"></div>'
    renderHook(() => useCommandShortcuts())
    fireEvent.keyDown(window, { key: 'b', ctrlKey: true })
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(false)
  })

  it('ignores bare-key shortcuts typed into a text field', () => {
    usePageStore.setState({ page: { path: 'content/a.md' } as never })
    const open = vi.spyOn(useUiStore.getState(), 'openRename')
    useUiStore.setState({ openRename: open })
    document.body.innerHTML = '<input id="t" />'
    const input = document.getElementById('t')!
    const e = new KeyboardEvent('keydown', { key: 'F2', bubbles: true, cancelable: true })
    input.dispatchEvent(e)
    expect(dispatchShortcut(e, buildCommands())).toBe(false)
    expect(open).not.toHaveBeenCalled()
    usePageStore.setState({ page: null })
  })

  it('still runs Mod shortcuts typed into a text field', () => {
    document.body.innerHTML = '<input id="t" />'
    renderHook(() => useCommandShortcuts())
    fireEvent.keyDown(document.getElementById('t')!, { key: 'b', ctrlKey: true })
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(true)
  })
})

describe('layout persistence', () => {
  it('persists the collapsed state and size', () => {
    useLayoutStore.getState().toggleSidebar()
    useLayoutStore.getState().setSidebarSize(30)
    expect(JSON.parse(localStorage.getItem('studio.layout') ?? '{}')).toMatchObject({
      sidebarSize: 30,
      sidebarCollapsed: true,
    })
  })
})
