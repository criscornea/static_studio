import { describe, expect, it } from 'vitest'

import { useLayoutStore } from '../layout'

describe('layout store sidebar panel', () => {
  it('defaults to files, switches panel, expands and persists', () => {
    expect(useLayoutStore.getState().sidebarPanel).toBe('files')

    useLayoutStore.setState({ sidebarCollapsed: true })
    useLayoutStore.getState().showSidebarPanel('search')

    const s = useLayoutStore.getState()
    expect(s.sidebarPanel).toBe('search')
    expect(s.sidebarCollapsed).toBe(false)
    expect(JSON.parse(localStorage.getItem('studio.layout') ?? '{}').sidebarPanel).toBe('search')

    s.showSidebarPanel('files')
  })
})
