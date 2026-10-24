import { fireEvent, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import { useLayoutStore } from '@/stores/layout'
import { useSidebarShortcut } from '../useSidebarShortcut'

beforeEach(() => {
  localStorage.clear()
  useLayoutStore.setState({ sidebarCollapsed: false, sidebarSize: 22 })
})

describe('sidebar toggle', () => {
  it('toggles on Ctrl+B and Cmd+B', () => {
    renderHook(() => useSidebarShortcut())
    fireEvent.keyDown(window, { key: 'b', ctrlKey: true })
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(true)
    fireEvent.keyDown(window, { key: 'b', metaKey: true })
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(false)
  })

  it('ignores a plain B', () => {
    renderHook(() => useSidebarShortcut())
    fireEvent.keyDown(window, { key: 'b' })
    expect(useLayoutStore.getState().sidebarCollapsed).toBe(false)
  })

  it('persists the collapsed state and size', () => {
    useLayoutStore.getState().toggleSidebar()
    useLayoutStore.getState().setSidebarSize(30)
    expect(JSON.parse(localStorage.getItem('studio.layout') ?? '{}')).toMatchObject({
      sidebarSize: 30,
      sidebarCollapsed: true,
    })
  })
})
