import { useId, useState } from 'react'

import { api } from '@/api/client'
import { ApiError, asApiError } from '@/api/types'
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
import { dirOf } from '@/lib/tree'
import { cancelAutosave, usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'

interface RenamePageDialogProps {
  // Full path of the page to rename, or null when the dialog is closed.
  path: string | null
  onClose: () => void
}

export function RenamePageDialog({ path, onClose }: RenamePageDialogProps) {
  return (
    <Dialog open={path !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        {/* Mounted only while open, so every opening starts with a fresh form. */}
        {path !== null && <RenameForm path={path} onDone={onClose} />}
      </DialogContent>
    </Dialog>
  )
}

// A path that starts with a content dir moves the page into that dir; anything
// else stays inside the page's current dir.
function targetPath(value: string, prefix: string, dirs: string[]): string {
  const v = value.trim().replace(/^\/+/, '')
  return dirs.some((d) => v.startsWith(d + '/')) ? v : prefix + v
}

function RenameForm({ path, onDone }: { path: string; onDone: () => void }) {
  const primaryDir = useProjectStore((s) => s.project?.contentDir ?? 'content')
  const dirInfos = useProjectStore((s) => s.project?.contentDirs)
  const loadTree = useProjectStore((s) => s.loadTree)
  const openPath = usePageStore((s) => s.path)
  const load = usePageStore((s) => s.load)

  const allDirs = dirInfos?.length ? dirInfos.map((d) => d.path) : [primaryDir]
  const contentDir = dirOf(path, allDirs) ?? primaryDir
  const prefix = contentDir + '/'
  const relative = path.startsWith(prefix) ? path.slice(prefix.length) : path
  const [value, setValue] = useState(relative)
  const [error, setError] = useState<ApiError | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const id = useId()

  async function submit() {
    setSubmitting(true)
    setError(null)
    let next: string
    try {
      if (openPath === path) {
        cancelAutosave()
        await usePageStore.getState().save()
        const st = usePageStore.getState()
        if (st.dirty || st.saveError) {
          setError(
            st.saveError ??
              new ApiError('cannot_save_page', 'Save the page before renaming it.', 0),
          )
          setSubmitting(false)
          return
        }
      }
      const res = await api.renamePage({
        from: path,
        to: targetPath(value, prefix, allDirs),
      })
      next = res.path
    } catch (err) {
      setError(asApiError(err))
      setSubmitting(false)
      return
    }
    onDone()
    try {
      await loadTree()
      if (openPath === path) {
        await load(next)
      }
    } catch (err) {
      console.error('Page renamed, but refreshing failed:', asApiError(err).message)
    }
  }

  const unchanged = value.trim() === relative

  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault()
        void submit()
      }}
    >
      <DialogHeader>
        <DialogTitle>Rename or move page</DialogTitle>
        <DialogDescription>
          Change the file name, or add a folder such as posts/ to move it. Missing folders are
          created.
          {allDirs.length > 1 && ' Start the path with another content folder to move it there.'}
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-1.5">
        <Label htmlFor={id}>Path inside {contentDir}</Label>
        <Input
          id={id}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          autoFocus
          autoComplete="off"
          spellCheck={false}
          className="font-mono"
        />
        {allDirs.length > 1 && (
          <p className="text-xs text-muted-foreground">
            Will be saved as <span className="font-mono">{targetPath(value, prefix, allDirs)}</span>
          </p>
        )}
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error.message}</AlertDescription>
        </Alert>
      )}

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onDone}>
          Cancel
        </Button>
        <Button type="submit" disabled={submitting || value.trim() === '' || unchanged}>
          {submitting ? 'Renaming…' : 'Rename'}
        </Button>
      </DialogFooter>
    </form>
  )
}
