import { useEffect } from 'react'

import { ContentTree } from '@/components/ContentTree'
import { PageView } from '@/components/PageView'
import { ProjectPicker } from '@/components/ProjectPicker'
import { Button } from '@/components/ui/button'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import { ScrollArea } from '@/components/ui/scroll-area'
import { usePageStore } from '@/stores/page'
import { selectIsOpen, useProjectStore } from '@/stores/project'

export default function App() {
  const isOpen = useProjectStore(selectIsOpen)
  const root = useProjectStore((s) => s.project?.root)
  const tree = useProjectStore((s) => s.tree)
  const close = useProjectStore((s) => s.close)
  const restore = useProjectStore((s) => s.restore)
  const selected = usePageStore((s) => s.path)
  const load = usePageStore((s) => s.load)

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
        <Button variant="ghost" size="sm" onClick={() => void close()}>
          Close
        </Button>
      </header>

      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize="22%" minSize="12%" maxSize="45%">
          <ScrollArea className="h-full">
            <nav className="p-2">
              {tree?.children?.length ? (
                <ul>
                  {tree.children.map((child) => (
                    <ContentTree
                      key={child.path}
                      node={child}
                      selected={selected}
                      onSelect={(path) => void load(path)}
                    />
                  ))}
                </ul>
              ) : (
                <p className="text-sm text-muted-foreground">No content files yet.</p>
              )}
            </nav>
          </ScrollArea>
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
    </div>
  )
}
