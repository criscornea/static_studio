import { useEffect, useId, useState } from 'react'

import { Upload } from 'lucide-react'

import { api } from '@/api/client'
import { asApiError, type PublishSettings } from '@/api/types'
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
import { useUiStore } from '@/stores/ui'

/** The header button that opens the publish dialog. */
export function PublishButton() {
  const setOpen = useUiStore((s) => s.setPublishOpen)

  return (
    <IconButton label="Publish" shortcut="Ctrl+Shift+U" onClick={() => setOpen(true)}>
      <Upload />
    </IconButton>
  )
}

/** The publish dialog. Rendered once at the app level so a command can open it. */
export function PublishDialog() {
  const open = useUiStore((s) => s.publishOpen)
  const setOpen = useUiStore((s) => s.setPublishOpen)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent>
        <PublishForm />
      </DialogContent>
    </Dialog>
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
  const targetID = useId()
  const destID = useId()
  const portID = useId()
  const identityID = useId()
  const status = usePublishStore((s) => s.status)
  const lines = usePublishStore((s) => s.lines)
  const pubError = usePublishStore((s) => s.error)
  const start = usePublishStore((s) => s.start)
  const cancel = usePublishStore((s) => s.cancel)
  const attach = usePublishStore((s) => s.attach)

  const [target, setTarget] = useState<PublishSettings['target']>('github-pages')
  const [remote, setRemote] = useState('')
  const [branch, setBranch] = useState('gh-pages')
  const [dest, setDest] = useState('')
  const [port, setPort] = useState('')
  const [identity, setIdentity] = useState('')
  const [del, setDel] = useState(false)
  const [dryRun, setDryRun] = useState(false)
  const [confirmDel, setConfirmDel] = useState(false)
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
          apply(s)
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

  function apply(s: PublishSettings) {
    setTarget(s.target === 'rsync' ? 'rsync' : 'github-pages')
    setRemote(s.remote)
    setBranch(s.branch)
    setDest(s.rsyncDest ?? '')
    setPort(s.rsyncPort ? String(s.rsyncPort) : '')
    setIdentity(s.rsyncIdentity ?? '')
    setDel(s.rsyncDelete ?? false)
    setDryRun(s.rsyncDryRun ?? false)
  }

  const running = status === 'running'
  const isRsync = target === 'rsync'
  const ready = isRsync ? dest.trim() !== '' : remote.trim() !== ''

  function toggleDelete(on: boolean) {
    if (on) {
      setConfirmDel(true)
    } else {
      setDel(false)
      setConfirmDel(false)
    }
  }

  async function publish() {
    setError(null)
    try {
      const portNum = port.trim() === '' ? 0 : Number(port)
      apply(
        await api.savePublishSettings({
          target,
          remote,
          branch,
          rsyncDest: dest,
          rsyncPort: Number.isInteger(portNum) ? portNum : -1,
          rsyncIdentity: identity,
          rsyncDelete: del,
          rsyncDryRun: dryRun,
        }),
      )
    } catch (err) {
      setError(asApiError(err).message)
      return
    }
    await start()
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>{isRsync ? 'Publish with rsync' : 'Publish to GitHub Pages'}</DialogTitle>
        <DialogDescription>
          {isRsync
            ? 'Builds the site and copies it with rsync over SSH. Your own SSH key, agent and known hosts sign in; nothing is stored here.'
            : 'Builds the site and pushes it to a branch of your git remote. Your own SSH key or git credential helper signs in; nothing is stored here. The branch is replaced on every publish.'}
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-3">
        <div className="space-y-1">
          <Label htmlFor={targetID}>Target</Label>
          <select
            id={targetID}
            value={target}
            disabled={!loaded || running}
            className="h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm"
            onChange={(e) => setTarget(e.target.value === 'rsync' ? 'rsync' : 'github-pages')}
          >
            <option value="github-pages">GitHub Pages (git)</option>
            <option value="rsync">rsync over SSH</option>
          </select>
        </div>
        {isRsync ? (
          <>
            <div className="space-y-1">
              <Label htmlFor={destID}>Destination</Label>
              <Input
                id={destID}
                value={dest}
                disabled={!loaded || running}
                placeholder="user@example.com:/var/www/site"
                onChange={(e) => setDest(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor={portID}>Port</Label>
              <Input
                id={portID}
                value={port}
                inputMode="numeric"
                disabled={!loaded || running}
                placeholder="22"
                onChange={(e) => setPort(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor={identityID}>Identity file (optional)</Label>
              <Input
                id={identityID}
                value={identity}
                disabled={!loaded || running}
                placeholder="/home/you/.ssh/id_ed25519"
                onChange={(e) => setIdentity(e.target.value)}
              />
            </div>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={del || confirmDel}
                disabled={!loaded || running}
                onChange={(e) => toggleDelete(e.target.checked)}
              />
              Delete files on the destination that are not in the site
            </label>
            {confirmDel && (
              <Alert variant="destructive">
                <AlertDescription>
                  <p>
                    Delete removes every file on the destination that is not in your built site.
                    This cannot be undone. Try a dry run first.
                  </p>
                  <div className="mt-2 flex gap-2">
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => {
                        setDel(true)
                        setConfirmDel(false)
                      }}
                    >
                      Enable delete
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => setConfirmDel(false)}>
                      Keep it off
                    </Button>
                  </div>
                </AlertDescription>
              </Alert>
            )}
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={dryRun}
                disabled={!loaded || running}
                onChange={(e) => setDryRun(e.target.checked)}
              />
              Dry run (show what would change, copy nothing)
            </label>
          </>
        ) : (
          <>
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
          </>
        )}
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
          <Button disabled={!loaded || !ready || confirmDel} onClick={() => void publish()}>
            {isRsync && dryRun ? 'Publish (dry run)' : 'Publish'}
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
