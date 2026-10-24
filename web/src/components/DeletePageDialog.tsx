import { useState } from 'react'

import { api } from '@/api/client'
import { asApiError, type ApiError } from '@/api/types'
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
import { usePageStore } from '@/stores/page'
import { useProjectStore } from '@/stores/project'

interface DeletePageDialogProps {
  // Full path of the page to delete, or null when the dialog is closed.
  path: string | null
  onClose: () => void
}

export function DeletePageDialog({ path, onClose }: DeletePageDialogProps) {
  return (
    <Dialog open={path !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        {path !== null && <DeleteForm path={path} onDone={onClose} />}
      </DialogContent>
    </Dialog>
  )
}

function DeleteForm({ path, onDone }: { path: string; onDone: () => void }) {
  const loadTree = useProjectStore((s) => s.loadTree)
  const openPath = usePageStore((s) => s.path)
  const clear = usePageStore((s) => s.clear)

  const [error, setError] = useState<ApiError | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function submit() {
    setSubmitting(true)
    setError(null)
    try {
      await api.deletePage(path)
    } catch (err) {
      setError(asApiError(err))
      setSubmitting(false)
      return
    }
    if (openPath === path) {
      clear()
    }
    onDone()
    try {
      await loadTree()
    } catch (err) {
      console.error('Page deleted, but reloading the tree failed:', asApiError(err).message)
    }
  }

  return (
    <div className="space-y-4">
      <DialogHeader>
        <DialogTitle>Delete page?</DialogTitle>
        <DialogDescription>
          <span className="font-mono">{path}</span> will be deleted from disk. This cannot be
          undone.
        </DialogDescription>
      </DialogHeader>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error.message}</AlertDescription>
        </Alert>
      )}

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onDone}>
          Cancel
        </Button>
        <Button
          type="button"
          variant="destructive"
          disabled={submitting}
          onClick={() => void submit()}
        >
          {submitting ? 'Deleting…' : 'Delete'}
        </Button>
      </DialogFooter>
    </div>
  )
}
