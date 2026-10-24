import { useEffect, useState } from 'react'
import { api } from '@/api/client'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { FrontmatterForm } from '@/components/FrontmatterForm'
import { MarkdownEditor } from '@/components/MarkdownEditor'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'
import { usePageStore } from '@/stores/page'

function formatLabel(format: string | undefined): string {
  switch (format) {
    case 'yaml':
      return 'YAML'
    case 'toml':
      return 'TOML'
    default:
      return 'no frontmatter'
  }
}

function Meta({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <dt className="text-xs uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 text-sm">{children}</dd>
    </div>
  )
}

export function PageView() {
  const [uploadError, setUploadError] = useState<{ path: string | null; message: string } | null>(null)
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
  const saving = page !== null && savingPath === page.path
  const conflict = saveError?.code === 'page_conflict'
  const status = saving ? 'Saving…' : conflict ? 'Conflict' : dirty ? 'Unsaved changes' : 'Saved'

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

  // Ctrl/Cmd+S saves even when the editor is not focused.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's' && !e.defaultPrevented) {
        e.preventDefault()
        void save()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [save])

  if (path === null) {
    return <p className="text-sm text-muted-foreground">Select a page.</p>
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertDescription>{error.message}</AlertDescription>
      </Alert>
    )
  }

  if (page === null) {
    return <p className="text-sm text-muted-foreground">Loading…</p>
  }

  const unchanged = !dirty
  const date = formatDate(page.fields.date)

  return (
    <article className={cn(loading && 'opacity-50')}>
      <header>
        <h1 className="mb-4 text-2xl font-semibold">{page.fields.title || 'Untitled'}</h1>

        <dl className="mb-3 flex flex-wrap gap-x-8 gap-y-3">
          {date && <Meta label="Date">{date}</Meta>}
          <Meta label="Status">
            <Badge variant={page.fields.draft ? 'secondary' : 'default'}>
              {page.fields.draft ? 'Draft' : 'Published'}
            </Badge>
          </Meta>
          {page.fields.tags?.length ? (
            <Meta label="Tags">
              <span className="flex gap-1">
                {page.fields.tags.map((tag) => (
                  <Badge key={tag} variant="outline">
                    {tag}
                  </Badge>
                ))}
              </span>
            </Meta>
          ) : null}
          <Meta label="Format">{formatLabel(page.format)}</Meta>
        </dl>

        <p className="font-mono text-xs text-muted-foreground">{page.path}</p>
      </header>

      <div className="mt-6 border-t pt-6">
        {uploadMessage && (
          <Alert variant="destructive" className="mb-3">
            <AlertDescription>{uploadMessage}</AlertDescription>
          </Alert>
        )}
        {saveError && (
          <Alert variant="destructive" className="mb-3">
            <AlertDescription>{saveError.message}</AlertDescription>
          </Alert>
        )}
        <FrontmatterForm
          key={page.path}
          fields={fieldsDraft ?? page.fields}
          onChange={setFieldsDraft}
        />
        <div className="mb-2 flex items-center gap-3">
          <Button
            size="sm"
            onClick={() => void save()}
            disabled={saving || unchanged}
          >
            {saving ? 'Saving…' : 'Save'}
          </Button>
          <span className="text-xs text-muted-foreground" role="status">
            {status}
          </span>
          <span className="text-xs text-muted-foreground">Ctrl/Cmd+S</span>
        </div>
        <MarkdownEditor
          key={page.path}
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
      </div>
    </article>
  )
}
