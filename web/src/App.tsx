import { useEffect } from 'react'

import { BuildButton, BuildPanel } from '@/components/BuildPanel'
import { DevServerButton } from '@/components/DevServer'
import { PageView } from '@/components/PageView'
import { ProjectPicker } from '@/components/ProjectPicker'
import { Sidebar } from '@/components/Sidebar'
import { Button } from '@/components/ui/button'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import { ScrollArea } from '@/components/ui/scroll-area'
import { selectIsOpen, useProjectStore } from '@/stores/project'

export default function App() {
  const isOpen = useProjectStore(selectIsOpen)
  const root = useProjectStore((s) => s.project?.root)
  const close = useProjectStore((s) => s.close)
  const restore = useProjectStore((s) => s.restore)

  // The backend holds the open project, so a page reload reconnects to it.
  useEffect(() => {
    void restore()
  }, [restore])

  if (!isOpen) {
    return <ProjectPicker />
  }

  return (
    <div className="flex h-screen flex-col">
      <header className="flex items-center gap-4 border-b px-4 py-2">
        <span className="flex-1 truncate font-mono text-xs text-muted-foreground">{root}</span>
        <DevServerButton />
        <BuildButton />
        <Button variant="ghost" size="sm" onClick={() => void close()}>
          Close
        </Button>
      </header>

      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize="22%" minSize="12%" maxSize="45%">
          <Sidebar />
        </ResizablePanel>
        <ResizableHandle />
        <ResizablePanel>
          <ScrollArea className="h-full">
            <main className="p-8">
              <PageView />
            </main>
          </ScrollArea>
        </ResizablePanel>
      </ResizablePanelGroup>
      <BuildPanel />
    </div>
  )
}
