import { useEffect } from 'react'
import { Files, Search } from 'lucide-react'

import { AppDialogs } from '@/components/AppDialogs'
import { BuildPanel } from '@/components/BuildPanel'
import { EditorHeader } from '@/components/EditorHeader'
import { PageView } from '@/components/PageView'
import { ProjectPicker } from '@/components/ProjectPicker'
import { SearchPanel } from '@/components/SearchPanel'
import { Sidebar } from '@/components/Sidebar'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import { StatusBar } from '@/components/StatusBar'
import { ThemeButton } from '@/components/ThemeButton'
import { useCommandShortcuts } from '@/hooks/useCommandShortcuts'
import { useLayoutStore } from '@/stores/layout'
import { selectIsOpen, useProjectStore } from '@/stores/project'

export default function App() {
  const isOpen = useProjectStore(selectIsOpen)
  const root = useProjectStore((s) => s.project?.root)
  const close = useProjectStore((s) => s.close)
  const restore = useProjectStore((s) => s.restore)

  const collapsed = useLayoutStore((s) => s.sidebarCollapsed)
  const sidebarSize = useLayoutStore((s) => s.sidebarSize)
  const panel = useLayoutStore((s) => s.sidebarPanel)
  const showPanel = useLayoutStore((s) => s.showSidebarPanel)
  const toggleSidebar = useLayoutStore((s) => s.toggleSidebar)
  const setSidebarSize = useLayoutStore((s) => s.setSidebarSize)
  const buildLogOpen = useLayoutStore((s) => s.buildLogOpen)
  const buildLogSize = useLayoutStore((s) => s.buildLogSize)
  const setBuildLogSize = useLayoutStore((s) => s.setBuildLogSize)
  useCommandShortcuts()

  // The backend holds the open project, so a page reload reconnects to it.
  useEffect(() => {
    void restore()
  }, [restore])

  if (!isOpen) {
    return <ProjectPicker />
  }

  return (
    <div className="grid h-full grid-cols-[2.5rem_minmax(0,1fr)] grid-rows-[2.25rem_minmax(0,1fr)_1.5rem] overflow-hidden">
      <header className="col-span-2 flex items-center gap-3 border-b bg-sidebar px-3">
        <span className="flex-1 truncate font-mono text-xs text-muted-foreground">{root}</span>
        <ThemeButton />
        <Button variant="ghost" size="sm" onClick={() => void close()}>
          Close
        </Button>
      </header>

      <nav
        aria-label="Activity bar"
        className="flex flex-col items-center gap-1 border-r bg-sidebar py-1"
      >
        <TooltipProvider>
          {(
            [
              { id: 'files', label: 'Files', Icon: Files },
              { id: 'search', label: 'Search', Icon: Search },
            ] as const
          ).map(({ id, label, Icon }) => {
            const active = !collapsed && panel === id
            return (
              <Tooltip key={id}>
                <TooltipTrigger asChild>
                  <Button
                    variant={active ? 'secondary' : 'ghost'}
                    size="icon"
                    aria-label={label}
                    aria-pressed={active}
                    onClick={() => (active ? toggleSidebar() : showPanel(id))}
                  >
                    <Icon />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="right">{label}</TooltipContent>
              </Tooltip>
            )
          })}
        </TooltipProvider>
      </nav>

      <div className="min-h-0 min-w-0">
        <ResizablePanelGroup
          orientation="horizontal"
          className="h-full"
          onLayoutChanged={(layout) => {
            const size = layout.sidebar
            if (size !== undefined) {
              setSidebarSize(size)
            }
          }}
        >
          {!collapsed && (
            <>
              <ResizablePanel
                id="sidebar"
                defaultSize={`${sidebarSize}%`}
                minSize="12%"
                maxSize="45%"
              >
                {panel === 'search' ? <SearchPanel /> : <Sidebar />}
              </ResizablePanel>
              <ResizableHandle />
            </>
          )}
          <ResizablePanel id="main">
            <ResizablePanelGroup
              orientation="vertical"
              className="h-full"
              onLayoutChanged={(layout) => {
                const size = layout.buildlog
                if (size !== undefined) {
                  setBuildLogSize(size)
                }
              }}
            >
              <ResizablePanel id="editorarea" minSize="20%">
                <div className="flex h-full min-h-0 flex-col">
                  <EditorHeader />
                  <main className="min-h-0 flex-1">
                    <PageView />
                  </main>
                </div>
              </ResizablePanel>
              {buildLogOpen && (
                <>
                  <ResizableHandle />
                  <ResizablePanel
                    id="buildlog"
                    defaultSize={`${buildLogSize}%`}
                    minSize="10%"
                    maxSize="80%"
                  >
                    <BuildPanel />
                  </ResizablePanel>
                </>
              )}
            </ResizablePanelGroup>
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>

      <StatusBar />
      <AppDialogs />
    </div>
  )
}
