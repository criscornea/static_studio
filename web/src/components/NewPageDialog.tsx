import { useEffect, useId, useMemo, useState } from 'react'

import { api } from '@/api/client'
import { asApiError, type ApiError, type Archetype } from '@/api/types'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { collectFolders, folderOf } from '@/lib/tree'
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'

interface NewPageDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  // Folder to preselect, relative to the content directory. Defaults to the open page's folder.
  folder?: string
}

export function NewPageDialog({ open, onOpenChange, folder }: NewPageDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        {/* Mounted only while open, so every opening starts with a fresh form. */}
        <NewPageForm initialFolder={folder} onDone={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  )
}

// Radix Select cannot use an empty string as a value.
const automatic = '__auto__'

function NewPageForm({ onDone, initialFolder }: { onDone: () => void; initialFolder?: string }) {
  const tree = useProjectStore((s) => s.tree)
  const contentDir = useProjectStore((s) => s.project?.contentDir ?? 'content')
  const loadTree = useProjectStore((s) => s.loadTree)
  const selected = usePageStore((s) => s.path)
  const load = usePageStore((s) => s.load)

  const [name, setName] = useState('')
  const [folder, setFolder] = useState(() => initialFolder ?? folderOf(selected, contentDir))
  const [archetype, setArchetype] = useState(automatic)
  const [archetypes, setArchetypes] = useState<Archetype[]>([])
  const [error, setError] = useState<ApiError | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const folders = useMemo(() => collectFolders(tree), [tree])
  const ids = { name: useId(), folder: useId(), archetype: useId(), folders: useId() }

  useEffect(() => {
    // Without the list the form still works: the backend picks the template.
    api.archetypes().then(setArchetypes, () => setArchetypes([]))
  }, [])

  async function submit() {
    setSubmitting(true)
    setError(null)
    try {
      const page = await api.createPage({
        name,
        folder,
        archetype: archetype === automatic ? '' : archetype,
      })
      await loadTree()
      await load(page.path)
      onDone()
    } catch (err) {
      setError(asApiError(err))
      setSubmitting(false)
    }
  }

  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault()
        void submit()
      }}
    >
      <DialogHeader>
        <DialogTitle>New page</DialogTitle>
        <DialogDescription>
          The file name is made from the name, so “My first post” becomes my-first-post.md.
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-1.5">
        <Label htmlFor={ids.name}>Name</Label>
        <Input
          id={ids.name}
          value={name}
          onChange={(e) => setName(e.target.value)}
          autoFocus
          autoComplete="off"
          placeholder="My first post"
        />
      </div>

      <div className="space-y-1.5">
        <Label htmlFor={ids.folder}>Folder</Label>
        <Input
          id={ids.folder}
          value={folder}
          onChange={(e) => setFolder(e.target.value)}
          list={ids.folders}
          autoComplete="off"
          spellCheck={false}
          placeholder={`${contentDir} (top level)`}
          className="font-mono"
        />
        <datalist id={ids.folders}>
          {folders.map((f) => (
            <option key={f} value={f} />
          ))}
        </datalist>
      </div>

      {archetypes.length > 0 && (
        <div className="space-y-1.5">
          <Label htmlFor={ids.archetype}>Template</Label>
          <Select value={archetype} onValueChange={setArchetype}>
            <SelectTrigger id={ids.archetype} className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={automatic}>Automatic (matches the folder)</SelectItem>
              {archetypes.map((a) => (
                <SelectItem key={a.name} value={a.name}>
                  {a.name}
                  {a.builtin ? ' (built in)' : ''}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error.message}</AlertDescription>
        </Alert>
      )}

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onDone}>
          Cancel
        </Button>
        <Button type="submit" disabled={submitting || name.trim() === ''}>
          {submitting ? 'Creating…' : 'Create page'}
        </Button>
      </DialogFooter>
    </form>
  )
}
