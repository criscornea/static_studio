import { fireEvent, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import { useLayoutStore } from '@/stores/layout'
import { useBuildLogShortcut } from '../useBuildLogShortcut'

beforeEach(() => {
  useLayoutStore.setState({ buildLogOpen: false })
})

describe('build log toggle', () => {
  it('toggles on Ctrl+J and Cmd+J', () => {
    renderHook(() => useBuildLogShortcut())
    fireEvent.keyDown(window, { key: 'j', ctrlKey: true })
    expect(useLayoutStore.getState().buildLogOpen).toBe(true)
    fireEvent.keyDown(window, { key: 'j', metaKey: true })
    expect(useLayoutStore.getState().buildLogOpen).toBe(false)
  })

  it('ignores a plain J', () => {
    renderHook(() => useBuildLogShortcut())
    fireEvent.keyDown(window, { key: 'j' })
    expect(useLayoutStore.getState().buildLogOpen).toBe(false)
  })
})
