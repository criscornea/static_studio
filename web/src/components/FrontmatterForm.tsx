import { useState } from 'react'
import { Pencil } from 'lucide-react'
import type { PageFields } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface Props {
  fields: PageFields
  onChange: (fields: PageFields) => void
}

interface InlineProps {
  value: string
  label: string
  placeholder?: string
  inputClassName?: string
  onCommit: (value: string) => void
  children: (startEdit: () => void) => React.ReactNode
}

/** Shows children until edited; Enter or blur commits, Escape cancels. */
function InlineEdit({ value, label, placeholder, inputClassName, onCommit, children }: InlineProps) {
  const [editing, setEditing] = useState(false)
  const [text, setText] = useState(value)

  if (!editing) {
    return (
      <>
        {children(() => {
          setText(value)
          setEditing(true)
        })}
      </>
    )
  }
  const finish = (commit: boolean) => {
    setEditing(false)
    if (commit && text !== value) {
      onCommit(text)
    }
  }
  return (
    <Input
      autoFocus
      aria-label={label}
      placeholder={placeholder}
      className={inputClassName}
      value={text}
      onChange={(e) => setText(e.target.value)}
      onBlur={() => finish(true)}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          e.preventDefault()
          finish(true)
        } else if (e.key === 'Escape') {
          e.preventDefault()
          finish(false)
        }
      }}
    />
  )
}

/**
 * Title, date, draft flag and a tags placeholder. Mount it with a key per page.
 */
export function FrontmatterForm({ fields, onChange }: Props) {
  return (
    <section aria-label="Properties" className="grid gap-1 border-b bg-sidebar/50 px-3 py-3">
      <div className="group flex min-w-0 items-center gap-1.5">
        <InlineEdit
          value={fields.title}
          label="Title"
          inputClassName="h-9 text-xl font-semibold md:text-xl"
          onCommit={(title) => onChange({ ...fields, title })}
        >
          {(startEdit) => (
            <button
              type="button"
              aria-label="Edit title"
              className="flex min-w-0 cursor-pointer items-center gap-1.5 rounded text-left text-xl font-semibold"
              onClick={startEdit}
            >
              <span className="cursor-pointer truncate">{fields.title || 'Untitled'}</span>
              <Pencil aria-hidden className="cursor-pointer size-3.5 shrink-0 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100" />
            </button>
          )}
        </InlineEdit>
      </div>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
        <div className="group flex items-center gap-1">
          <InlineEdit
            value={fields.date ?? ''}
            label="Date (RFC 3339)"
            placeholder="2026-02-01T09:30:00+01:00"
            inputClassName="h-7 w-64"
            onCommit={(date) => onChange({ ...fields, date: date || undefined })}
          >
            {(startEdit) => (
              <button
                type="button"
                aria-label="Edit date"
                className="flex cursor-pointer items-center gap-1 rounded text-muted-foreground"
                onClick={startEdit}
              >
                <span className="cursor-pointer">{fields.date || 'No date'}</span>
                <Pencil aria-hidden className="cursor-pointer size-3 shrink-0 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100" />
              </button>
            )}
          </InlineEdit>
        </div>
        <div className="flex items-center gap-2">
          <input
            id="fm-draft"
            type="checkbox"
            checked={fields.draft}
            onChange={(e) => onChange({ ...fields, draft: e.target.checked })}
          />
          <Label htmlFor="fm-draft">Draft</Label>
        </div>
        <Button type="button" variant="outline" size="sm" className="h-7">
          Tags
        </Button>
      </div>
    </section>
  )
}
