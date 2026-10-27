import { useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { FileText, Pencil } from 'lucide-react'
import type { PageFields } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { Calendar } from '@/components/ui/calendar'

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
  /** Extra UI shown under the input while editing; setValue commits and closes. */
  extra?: (text: string, setValue: (value: string) => void) => React.ReactNode
  children: (startEdit: () => void) => React.ReactNode
}

const DATE_PREFIX = /^\s*(\d{4})-(\d{2})-(\d{2})/

function parseDay(text: string): Date | undefined {
  const m = DATE_PREFIX.exec(text)
  if (!m) return undefined
  const d = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]))
  return d.getMonth() === Number(m[2]) - 1 ? d : undefined
}

const pad = (n: number, w = 2) => String(n).padStart(w, '0')

/** Replaces the date part, keeping any time-of-day and timezone suffix. */
function withDay(text: string, day: Date): string {
  const ymd = `${pad(day.getFullYear(), 4)}-${pad(day.getMonth() + 1)}-${pad(day.getDate())}`
  return DATE_PREFIX.test(text) ? text.replace(DATE_PREFIX, ymd) : ymd
}

/** Shows children until edited; Enter or blur commits, Escape cancels. */
function InlineEdit({ value, label, placeholder, inputClassName, onCommit, extra, children }: InlineProps) {
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
  const finish = (commit: boolean, next = text) => {
    setEditing(false)
    if (commit && next !== value) {
      onCommit(next)
    }
  }
  const input = (
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
  if (!extra) return input
  return (
    <div className="relative flex flex-col items-start">
      {input}
      {/* Keep focus in the input so clicking the calendar does not blur-commit. */}
      <FloatingPanel>{extra(text, (v) => finish(true, v))}</FloatingPanel>
    </div>
  )
}

/**
 * Renders children in a body portal, fixed under its parent, so ancestors with
 * overflow clipping neither clip it nor grow around it.
 */
function FloatingPanel({ children }: { children: React.ReactNode }) {
  const anchor = useRef<HTMLSpanElement>(null)
  const [pos, setPos] = useState<{ left: number; top: number } | null>(null)
  useLayoutEffect(() => {
    const parent = anchor.current?.parentElement
    if (!parent) return
    const update = () => {
      const r = parent.getBoundingClientRect()
      setPos({ left: r.left, top: r.bottom + 4 })
    }
    update()
    window.addEventListener('resize', update)
    window.addEventListener('scroll', update, true)
    return () => {
      window.removeEventListener('resize', update)
      window.removeEventListener('scroll', update, true)
    }
  }, [])
  return (
    <>
      <span ref={anchor} hidden />
      {pos &&
        createPortal(
          <div
            style={{ position: 'fixed', left: pos.left, top: pos.top }}
            className="z-50 rounded-md border bg-popover p-2 text-popover-foreground shadow-md"
            onMouseDown={(e) => e.preventDefault()}
          >
            {children}
          </div>,
          document.body,
        )}
    </>
  )
}

/**
 * Title, date, draft flag and a tags placeholder. Mount it with a key per page.
 */
export function FrontmatterForm({ fields, onChange }: Props) {
  return (
    <section aria-label="Properties" className="grid gap-1 border-b bg-sidebar/50 px-3 py-3">
      <div className="group flex min-w-0 items-center gap-1.5">
        <button
          type="button"
          aria-label="Draft"
          title="Draft"
          aria-pressed={fields.draft}
          className={cn(
            'shrink-0 cursor-pointer rounded',
            fields.draft ? 'text-yellow-500' : 'text-muted-foreground',
          )}
          onClick={() => onChange({ ...fields, draft: !fields.draft })}
        >
          <FileText aria-hidden className="size-5" />
        </button>
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
            extra={(text, setValue) => {
              const selected = parseDay(text)
              return (
                <Calendar
                  key={selected ? `${selected.getFullYear()}-${selected.getMonth()}` : 'none'}
                  mode="single"
                  selected={selected}
                  defaultMonth={selected}
                  onSelect={(day) => day && setValue(withDay(text, day))}
                />
              )
            }}
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
        <Button type="button" variant="outline" size="sm" className="h-7">
          Tags
        </Button>
      </div>
    </section>
  )
}
