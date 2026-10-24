import { useEffect } from 'react'

import { buildCommands, dispatchShortcut } from '@/lib/commands'
import { useProjectStore } from '@/stores/project'

/** The one global key handler: maps shortcuts to commands while a project is open. */
export function useCommandShortcuts() {
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (useProjectStore.getState().project === null) {
        return
      }
      dispatchShortcut(e, buildCommands())
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])
}
