import { useEffect } from 'react'

import { useLayoutStore } from '@/stores/layout'

/** Ctrl/Cmd+B toggles the sidebar, as in most editors. */
export function useSidebarShortcut() {
  const toggle = useLayoutStore((s) => s.toggleSidebar)

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && !e.altKey && !e.shiftKey && e.key.toLowerCase() === 'b') {
        e.preventDefault()
        toggle()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [toggle])
}
