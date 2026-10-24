import { useEffect } from 'react'
import { Files } from 'lucide-react'

import { BuildButton, BuildPanel } from '@/components/BuildPanel'
import { DevServerButton } from '@/components/DevServer'
import { PublishButton } from '@/components/PublishDialog'
import { PageView } from '@/components/PageView'
import { ProjectPicker } from '@/components/ProjectPicker'
import { Sidebar } from '@/components/Sidebar'
import { Button } from '@/components/ui/button'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import { ScrollArea } from '@/components/ui/scroll-area'
import { StatusBar } from '@/components/StatusBar'
import { ThemeButton } from '@/components/ThemeButton'
import { useSidebarShortcut } from '@/hooks/useSidebarShortcut'
import { useLayoutStore } from '@/stores/layout'
import { selectIsOpen, useProjectStore } from '@/stores/project'

export default function App() {
  const isOpen = useProjectStore(selectIsOpen)
  const root = useProjectStore((s) => s.project?.root)
  const close = useProjectStore((s) => s.close)
  const restore = useProjectStore((s) => s.restore)

  const collapsed = useLayoutStore((s) => s.sidebarCollapsed)
  const sidebarSize = useLayoutStore((s) => s.sidebarSize)
  const toggleSidebar = useLayoutStore((s) => s.toggleSidebar)
  const setSidebarSize = useLayoutStore((s) => s.setSidebarSize)
  useSidebarShortcut()

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
        <DevServerButton />
        <BuildButton />
        <PublishButton />
        <ThemeButton />
        <Button variant="ghost" size="sm" onClick={() => void close()}>
          Close
        </Button>
      </header>

      <nav
        aria-label="Activity bar"
        className="flex flex-col items-center gap-1 border-r bg-sidebar py-1"
      >
        <Button
          variant={collapsed ? 'ghost' : 'secondary'}
          size="icon"
          aria-label={collapsed ? 'Show sidebar' : 'Hide sidebar'}
          aria-pressed={!collapsed}
          title="Toggle sidebar (Ctrl+B)"
          onClick={toggleSidebar}
        >
          <Files />
        </Button>
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
              <ResizablePanel id="sidebar" defaultSize={`${sidebarSize}%`} minSize="12%" maxSize="45%">
                <Sidebar />
              </ResizablePanel>
              <ResizableHandle />
            </>
          )}
          <ResizablePanel id="main">
            <div className="flex h-full min-h-0 flex-col">
              <ScrollArea className="min-h-0 flex-1">
                <main className="p-6">
                  <PageView />
                </main>
              </ScrollArea>
              <BuildPanel />
            </div>
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>

      <StatusBar />
    </div>
  )
}
