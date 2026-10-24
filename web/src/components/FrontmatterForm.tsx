import { useState } from 'react'
import type { PageFields } from '@/api/types'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface Props {
  fields: PageFields
  onChange: (fields: PageFields) => void
}

function parseTags(text: string): string[] {
  return text
    .split(',')
    .map((t) => t.trim())
    .filter((t) => t !== '')
}

/**
 * Edits the known frontmatter fields. Mount it with a key per page: the
 * tags text is held locally so typing a trailing comma is not swallowed.
 */
export function FrontmatterForm({ fields, onChange }: Props) {
  const [tagsText, setTagsText] = useState((fields.tags ?? []).join(', '))

  return (
    <form className="mb-4 grid gap-3 sm:grid-cols-2" onSubmit={(e) => e.preventDefault()}>
      <div className="grid gap-1 sm:col-span-2">
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
      <div className="flex items-center gap-2 sm:col-span-2">
        <input
          id="fm-draft"
          type="checkbox"
          checked={fields.draft}
          onChange={(e) => onChange({ ...fields, draft: e.target.checked })}
        />
        <Label htmlFor="fm-draft">Draft</Label>
      </div>
    </form>
  )
}
