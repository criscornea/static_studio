import { useEffect, useState } from 'react'
import { FileText } from 'lucide-react'

import { api } from '@/api/client'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { EmptyState, SkeletonRows } from '@/components/EmptyState'
import { PreviewFrame } from '@/components/DevServer'
import { FrontmatterForm } from '@/components/FrontmatterForm'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { cn } from '@/lib/utils'
import { useLayoutStore } from '@/stores/layout'
import { useBuildStore } from '@/stores/build'
import { usePageStore } from '@/stores/page'
import { useUiStore } from '@/stores/ui'

export function PageView() {
  const previewing = useLayoutStore((s) => s.previewOpen)
  const previewSize = useLayoutStore((s) => s.previewSize)
  const setPreviewSize = useLayoutStore((s) => s.setPreviewSize)
  const [uploadError, setUploadError] = useState<{ path: string | null; message: string } | null>(
    null,
  )
  const path = usePageStore((s) => s.path)
  // Errors belong to the page they happened on.
  const uploadMessage = uploadError?.path === path ? uploadError.message : null
  const page = usePageStore((s) => s.page)
  const loading = usePageStore((s) => s.loading)
  const error = usePageStore((s) => s.error)
  const draft = usePageStore((s) => s.draft)
  const fieldsDraft = usePageStore((s) => s.fieldsDraft)
  const setFieldsDraft = usePageStore((s) => s.setFieldsDraft)
  const savingPath = usePageStore((s) => s.savingPath)
  const saveError = usePageStore((s) => s.saveError)
  const setDraft = usePageStore((s) => s.setDraft)
  const save = usePageStore((s) => s.save)
  const dirty = usePageStore((s) => s.dirty)
  const setQuickOpen = useUiStore((s) => s.setQuickOpenOpen)
  const saving = page !== null && savingPath === page.path
  const jump = useBuildStore((s) => s.jump)
  // A build error on this page: a body line for the editor, else a frontmatter line.
  const bodyJump = jump && page && jump.file === page.path ? jump.line - page.bodyLine + 1 : null
  const editorLine = bodyJump !== null && bodyJump >= 1 ? bodyJump : null
  const frontmatterLine =
    jump && page && jump.file === page.path && editorLine === null ? jump.line : null

  // Warn before closing the tab while edits are unsaved or saving.
  useEffect(() => {
    if (!dirty && !saving) {
      return
    }
    const onUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', onUnload)
    return () => window.removeEventListener('beforeunload', onUnload)
  }, [dirty, saving])

  if (path === null) {
    return (
      <EmptyState
        icon={FileText}
        className="h-full"
        text="Select a page to start editing."
        action={{ label: 'Open page...', onClick: () => setQuickOpen(true) }}
      />
    )
  }

  if (error) {
    return (
      <Alert variant="destructive" className="m-6 w-auto">
        <AlertDescription>{error.message}</AlertDescription>
      </Alert>
    )
  }

  if (page === null) {
    return <SkeletonRows count={8} label="Loading page" className="p-6" />
  }

  return (
    <article className={cn('flex h-full min-h-0 flex-col', loading && 'opacity-50')}>
      <div className="max-h-[60%] shrink-0 overflow-auto">
        {uploadMessage && (
          <Alert variant="destructive" className="m-3 w-auto">
            <AlertDescription>{uploadMessage}</AlertDescription>
          </Alert>
        )}
        {saveError && (
          <Alert variant="destructive" className="m-3 w-auto">
            <AlertDescription>{saveError.message}</AlertDescription>
          </Alert>
        )}
        <FrontmatterForm
          key={`form-${page.path}`}
          fields={fieldsDraft ?? page.fields}
          format={page.format}
          onChange={setFieldsDraft}
        />
        {frontmatterLine !== null && (
          <Alert variant="destructive" className="m-3 w-auto">
            <AlertDescription>
              The build reported an error in the frontmatter at line {frontmatterLine}.
            </AlertDescription>
          </Alert>
        )}
      </div>
      <ResizablePanelGroup
        orientation="horizontal"
        className="min-h-0 flex-1"
        onLayoutChanged={(layout) => {
          const size = layout.preview
          if (size !== undefined) {
            setPreviewSize(size)
          }
        }}
      >
        <ResizablePanel id="editor" minSize="25%">
          <MarkdownEditor
            key={`editor-${page.path}`}
            jumpLine={editorLine}
            jumpToken={jump}
            value={draft ?? page.body}
            onChange={setDraft}
            onSave={() => void save()}
            onUploadImage={async (file) => {
              setUploadError(null)
              return (await api.uploadAsset(file)).path
            }}
            onUploadError={(err) =>
              setUploadError({ path, message: err instanceof Error ? err.message : String(err) })
            }
          />
        </ResizablePanel>
        {previewing && (
          <>
            <ResizableHandle />
            <ResizablePanel
              id="preview"
              defaultSize={`${previewSize}%`}
              minSize="20%"
              maxSize="75%"
            >
              <PreviewFrame key={`preview-${page.path}`} path={page.path} />
            </ResizablePanel>
          </>
        )}
      </ResizablePanelGroup>
    </article>
  )
}
