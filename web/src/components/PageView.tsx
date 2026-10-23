import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
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
  const path = usePageStore((s) => s.path)
  const page = usePageStore((s) => s.page)
  const loading = usePageStore((s) => s.loading)
  const error = usePageStore((s) => s.error)

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

      <pre className="mt-6 whitespace-pre-wrap border-t pt-6 font-mono text-sm leading-relaxed">
        {page.body}
      </pre>
    </article>
  )
}
