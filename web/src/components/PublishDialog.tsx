import { useEffect, useId, useState } from 'react'

import { Upload } from 'lucide-react'

import { api } from '@/api/client'
import { asApiError } from '@/api/types'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { IconButton } from '@/components/IconButton'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { usePublishStore } from '@/stores/publish'

/** The header button that opens the publish dialog. */
export function PublishButton() {
  const [open, setOpen] = useState(false)

  return (
    <>
      <IconButton label="Publish" onClick={() => setOpen(true)}>
        <Upload />
      </IconButton>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <PublishForm />
        </DialogContent>
      </Dialog>
    </>
  )
}

const statusText = {
  idle: '',
  running: 'Publishing...',
  succeeded: 'Published.',
  failed: 'Publishing failed.',
} as const

function PublishForm() {
  const remoteID = useId()
  const branchID = useId()
  const status = usePublishStore((s) => s.status)
  const lines = usePublishStore((s) => s.lines)
  const pubError = usePublishStore((s) => s.error)
  const start = usePublishStore((s) => s.start)
  const cancel = usePublishStore((s) => s.cancel)
  const attach = usePublishStore((s) => s.attach)

  const [remote, setRemote] = useState('')
  const [branch, setBranch] = useState('gh-pages')
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    attach()
  }, [attach])

  useEffect(() => {
    let alive = true
    api
      .publishSettings()
      .then((s) => {
        if (alive) {
          setRemote(s.remote)
          setBranch(s.branch)
          setLoaded(true)
        }
      })
      .catch((err: unknown) => {
        if (alive) {
          setError(asApiError(err).message)
          setLoaded(true)
        }
      })
    return () => {
      alive = false
    }
  }, [])

  const running = status === 'running'

  async function publish() {
    setError(null)
    try {
      const saved = await api.savePublishSettings({ remote, branch })
      setRemote(saved.remote)
      setBranch(saved.branch)
    } catch (err) {
      setError(asApiError(err).message)
      return
    }
    await start()
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>Publish to GitHub Pages</DialogTitle>
        <DialogDescription>
          Builds the site and pushes it to a branch of your git remote. Your own SSH key or git
          credential helper signs in; nothing is stored here. The branch is replaced on every
          publish.
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-3">
        <div className="space-y-1">
          <Label htmlFor={remoteID}>Remote</Label>
          <Input
            id={remoteID}
            value={remote}
            disabled={!loaded || running}
            placeholder="git@github.com:you/you.github.io.git"
            onChange={(e) => setRemote(e.target.value)}
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor={branchID}>Branch</Label>
          <Input
            id={branchID}
            value={branch}
            disabled={!loaded || running}
            onChange={(e) => setBranch(e.target.value)}
          />
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="flex items-center gap-2">
        <span role="status" className="flex-1 text-sm">
          {statusText[status]}
          {pubError && <span className="ml-2 text-destructive">{pubError}</span>}
        </span>
        {running ? (
          <Button variant="ghost" onClick={() => void cancel()}>
            Cancel
          </Button>
        ) : (
          <Button disabled={!loaded || remote.trim() === ''} onClick={() => void publish()}>
            Publish
          </Button>
        )}
      </div>

      {status !== 'idle' && (
        <pre
          aria-label="Publish output"
          className="h-40 overflow-auto bg-muted px-3 py-2 font-mono text-xs whitespace-pre-wrap"
        >
          {lines.join('\n')}
        </pre>
      )}
    </>
  )
}
