import { useEffect } from 'react'

import { useLayoutStore } from '@/stores/layout'

/** Ctrl/Cmd+J toggles the build log panel, as the terminal panel in most editors. */
export function useBuildLogShortcut() {
  const toggle = useLayoutStore((s) => s.toggleBuildLog)

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && !e.altKey && !e.shiftKey && e.key.toLowerCase() === 'j') {
        e.preventDefault()
        toggle()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [toggle])
}
