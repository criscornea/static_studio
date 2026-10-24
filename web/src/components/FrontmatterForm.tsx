import { useId, useState } from 'react'
import { ChevronRight } from 'lucide-react'
import type { PageFields } from '@/api/types'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { useLayoutStore } from '@/stores/layout'

interface Props {
  fields: PageFields
  /** Frontmatter format of the page, shown in the collapsed summary. */
  format?: string
  onChange: (fields: PageFields) => void
}

function parseTags(text: string): string[] {
  return text
    .split(',')
    .map((t) => t.trim())
    .filter((t) => t !== '')
}

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

/**
 * Edits the known frontmatter fields in a collapsible panel. Mount it with a
 * key per page: the tags text is held locally so typing a trailing comma is
 * not swallowed. The fields stay mounted while collapsed so that state lives on.
 */
export function FrontmatterForm({ fields, format, onChange }: Props) {
  const [tagsText, setTagsText] = useState((fields.tags ?? []).join(', '))
  const open = useLayoutStore((s) => s.propertiesOpen)
  const toggle = useLayoutStore((s) => s.toggleProperties)
  const panelID = useId()

  return (
    <section aria-label="Properties" className="border-b bg-sidebar/50">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={panelID}
        onClick={toggle}
        className="flex w-full items-center gap-1.5 px-3 py-1.5 text-left text-xs"
      >
        <ChevronRight className={cn('size-3.5 transition-transform', open && 'rotate-90')} />
        <span className="font-medium">Properties</span>
        <span className="min-w-0 flex-1 truncate text-muted-foreground">
          {fields.title || 'Untitled'}
          {fields.draft ? ' · draft' : ''}
          {format !== undefined ? ` · ${formatLabel(format)}` : ''}
        </span>
      </button>
      <form
        id={panelID}
        hidden={!open}
        className="grid gap-2 px-3 pb-3 sm:grid-cols-[2fr_1.5fr_1.5fr_auto]"
        onSubmit={(e) => e.preventDefault()}
      >
        <div className="grid gap-1">
          <Label htmlFor="fm-title">Title</Label>
          <Input
            id="fm-title"
            value={fields.title}
            onChange={(e) => onChange({ ...fields, title: e.target.value })}
          />
        </div>
        <div className="grid gap-1">
          <Label htmlFor="fm-date">Date (RFC 3339)</Label>
          <Input
            id="fm-date"
            placeholder="2026-02-01T09:30:00+01:00"
            value={fields.date ?? ''}
            onChange={(e) => onChange({ ...fields, date: e.target.value || undefined })}
          />
        </div>
        <div className="grid gap-1">
          <Label htmlFor="fm-tags">Tags (comma separated)</Label>
          <Input
            id="fm-tags"
            value={tagsText}
            onChange={(e) => {
              setTagsText(e.target.value)
              onChange({ ...fields, tags: parseTags(e.target.value) })
            }}
          />
        </div>
        <div className="flex items-end gap-2 pb-1.5">
          <input
            id="fm-draft"
            type="checkbox"
            checked={fields.draft}
            onChange={(e) => onChange({ ...fields, draft: e.target.checked })}
          />
          <Label htmlFor="fm-draft">Draft</Label>
        </div>
      </form>
    </section>
  )
}
