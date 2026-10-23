import { useState } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useProjectStore } from '@/stores/project'

export function ProjectPicker() {
  const open = useProjectStore((s) => s.open)
  const loading = useProjectStore((s) => s.loading)
  const error = useProjectStore((s) => s.error)
  const [path, setPath] = useState('')

  const trimmed = path.trim()

  async function submit() {
    if (trimmed === '') {
      return
    }
    await open(trimmed)
  }

  return (
    <section className="mx-auto mt-16 max-w-[34rem] px-6">
      <h1 className="mb-1 text-2xl font-semibold">static_studio</h1>
      <p className="mb-6 text-muted-foreground">Open the folder that contains your website.</p>

      <form
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        <Input
          value={path}
          onChange={(e) => setPath(e.target.value)}
          placeholder="/path/to/your/site"
          spellCheck={false}
          autoComplete="off"
          disabled={loading}
          className="font-mono"
        />
        <Button type="submit" disabled={loading || trimmed === ''}>
          {loading ? 'Opening…' : 'Open'}
        </Button>
      </form>

      {error && (
        <Alert variant="destructive" className="mt-4">
          <AlertDescription>{error.message}</AlertDescription>
        </Alert>
      )}

      <p className="mt-8 text-sm text-muted-foreground">
        This is the folder with your <code>hugo.toml</code> in it.
      </p>
    </section>
  )
}
