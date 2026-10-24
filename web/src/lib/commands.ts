import { formatShortcuts, formatTitles } from '@/lib/editorFormat'
import type { FormatId } from '@/lib/formatState'
import { matchesShortcut, usesMod } from '@/lib/shortcuts'
import { useBuildStore } from '@/stores/build'
import { useDevServerStore } from '@/stores/devserver'
import { useEditorStore } from '@/stores/editor'
import { useEditorModeStore } from '@/stores/editorMode'
import { useLayoutStore } from '@/stores/layout'
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'
import { useThemeStore } from '@/stores/theme'
import { useUiStore } from '@/stores/ui'

export interface Command {
  id: string
  title: string
  /** Palette group heading. */
  group: string
  /** Shortcut string such as "Mod+Shift+P". */
  shortcut?: string
  /**
   * "editor": the shortcut only works inside the focused editor (CodeMirror binds
   * it), so the global handler ignores it and it may reuse a global key.
   */
  scope?: 'editor'
  /** False when the command cannot run now; it is then dimmed and skipped. */
  enabled: boolean
  run: () => void
}

/** Builds the command list from the current app state. Cheap, so call it when needed. */
export function buildCommands(): Command[] {
  const ui = useUiStore.getState()
  const layout = useLayoutStore.getState()
  const page = usePageStore.getState()
  const build = useBuildStore.getState()
  const dev = useDevServerStore.getState()
  const theme = useThemeStore.getState()
  const project = useProjectStore.getState()

  const hasPage = page.page !== null
  const pagePath = page.page?.path ?? null
  const running = dev.running

  const editor = useEditorStore.getState()
  const editorMode = useEditorModeStore.getState()
  const canFormat = hasPage && editor.view !== null
  const formats = (Object.keys(formatTitles) as FormatId[]).map(
    (id): Command => ({
      id: `format.${id}`,
      title: formatTitles[id],
      group: 'Format',
      shortcut: formatShortcuts[id],
      scope: formatShortcuts[id] ? 'editor' : undefined,
      enabled: canFormat,
      run: () => editor.format(id),
    }),
  )

  return [
    {
      id: 'palette',
      title: 'Show command palette',
      group: 'Navigate',
      shortcut: 'Mod+Shift+P',
      enabled: true,
      run: () => ui.setPaletteOpen(true),
    },
    {
      id: 'page.open',
      title: 'Open page...',
      group: 'Navigate',
      shortcut: 'Mod+P',
      enabled: true,
      run: () => ui.setQuickOpenOpen(true),
    },
    {
      id: 'search.focus',
      title: 'Search pages',
      group: 'Navigate',
      shortcut: 'Mod+K',
      enabled: true,
      run: () => {
        if (layout.sidebarCollapsed) {
          layout.toggleSidebar()
        }
        ui.requestSearchFocus()
      },
    },
    {
      id: 'page.new',
      title: 'New page',
      group: 'Page',
      shortcut: 'Mod+Alt+N',
      enabled: true,
      run: () => ui.openNewPage(),
    },
    {
      id: 'page.rename',
      title: 'Rename page',
      group: 'Page',
      shortcut: 'F2',
      enabled: hasPage,
      run: () => pagePath !== null && ui.openRename(pagePath),
    },
    {
      id: 'page.delete',
      title: 'Delete page',
      group: 'Page',
      enabled: hasPage,
      run: () => pagePath !== null && ui.openDelete(pagePath),
    },
    {
      id: 'page.save',
      title: 'Save page',
      group: 'Page',
      shortcut: 'Mod+S',
      enabled: hasPage && page.dirty,
      run: () => void page.save(),
    },
    {
      id: 'view.sidebar',
      title: 'Toggle sidebar',
      group: 'View',
      shortcut: 'Mod+B',
      enabled: true,
      run: layout.toggleSidebar,
    },
    {
      id: 'view.livepreview',
      title: editorMode.mode === 'live' ? 'Editor: show source' : 'Editor: show live preview',
      group: 'View',
      enabled: true,
      run: editorMode.toggle,
    },
    {
      id: 'view.preview',
      title: 'Toggle preview',
      group: 'View',
      shortcut: 'Mod+Alt+V',
      enabled: dev.ready,
      run: layout.togglePreview,
    },
    {
      id: 'view.buildlog',
      title: 'Toggle build log',
      group: 'View',
      shortcut: 'Mod+J',
      enabled: true,
      run: layout.toggleBuildLog,
    },
    {
      id: 'site.build',
      title: 'Build site',
      group: 'Site',
      shortcut: 'Mod+Shift+B',
      enabled: build.status !== 'running',
      run: () => {
        layout.setBuildLogOpen(true)
        void build.start()
      },
    },
    {
      id: 'site.publish',
      title: 'Publish site...',
      group: 'Site',
      shortcut: 'Mod+Shift+U',
      enabled: true,
      run: () => ui.setPublishOpen(true),
    },
    {
      id: 'server.toggle',
      title: running ? 'Stop preview server' : 'Start preview server',
      group: 'Site',
      shortcut: 'Mod+Alt+S',
      enabled: !dev.busy,
      run: () => void (running ? dev.stop() : dev.start()),
    },
    {
      id: 'theme.system',
      title: 'Theme: follow system',
      group: 'Theme',
      enabled: theme.choice !== 'system',
      run: () => theme.setChoice('system'),
    },
    {
      id: 'theme.light',
      title: 'Theme: light',
      group: 'Theme',
      enabled: theme.choice !== 'light',
      run: () => theme.setChoice('light'),
    },
    {
      id: 'theme.dark',
      title: 'Theme: dark',
      group: 'Theme',
      enabled: theme.choice !== 'dark',
      run: () => theme.setChoice('dark'),
    },
    {
      id: 'shortcuts',
      title: 'Keyboard shortcuts',
      group: 'Help',
      shortcut: 'Mod+/',
      enabled: true,
      run: () => ui.setShortcutsOpen(true),
    },
    {
      id: 'project.close',
      title: 'Close project',
      group: 'Project',
      shortcut: 'Mod+Alt+W',
      enabled: project.project !== null,
      run: () => void project.close(),
    },
    ...formats,
  ]
}

/** Case-insensitive match where every query word must appear in the title or group. */
export function filterCommands(commands: Command[], query: string): Command[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean)
  if (words.length === 0) {
    return commands
  }
  return commands.filter((c) => {
    const hay = `${c.group} ${c.title}`.toLowerCase()
    return words.every((w) => hay.includes(w))
  })
}

function isEditable(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false
  }
  return target.isContentEditable || target.closest('input, textarea, select, .cm-editor') !== null
}

/**
 * Finds the command for a key press and runs it. Returns true when the key was
 * handled. It leaves alone keys that something else already handled (CodeMirror
 * calls preventDefault on its own bindings), keys typed into a dialog, and bare
 * keys such as F2 while a text field has focus.
 */
export function dispatchShortcut(e: KeyboardEvent, commands: Command[]): boolean {
  if (e.defaultPrevented || e.isComposing) {
    return false
  }
  if (document.querySelector('[role="dialog"]') !== null) {
    return false
  }
  const command = commands.find(
    (c) => c.scope !== 'editor' && c.shortcut !== undefined && matchesShortcut(e, c.shortcut),
  )
  if (command?.shortcut === undefined) {
    return false
  }
  if (!usesMod(command.shortcut) && isEditable(e.target)) {
    return false
  }
  // Claim the key even when the command is off, so the browser's own action
  // (such as the save-page dialog on Ctrl+S) never fires.
  e.preventDefault()
  if (command.enabled) {
    command.run()
  }
  return true
}
