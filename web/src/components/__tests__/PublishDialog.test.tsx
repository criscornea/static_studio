import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { usePublishStore } from '@/stores/publish'
import { useUiStore } from '@/stores/ui'
import { PublishButton, PublishDialog } from '../PublishDialog'

let puts: unknown[] = []
let stored: Record<string, unknown> = {}

const base = {
  target: 'github-pages',
  remote: 'git@example.com:a/b.git',
  branch: 'gh-pages',
  rsyncDest: '',
  rsyncPort: 0,
  rsyncIdentity: '',
  rsyncDelete: false,
  rsyncDryRun: false,
}

beforeEach(() => {
  puts = []
  stored = {}
  useUiStore.getState().setPublishOpen(false)
  usePublishStore.getState().reset()
  vi.stubGlobal(
    'EventSource',
    class {
      onmessage = null
      onerror = null
      close() {}
    },
  )
  vi.stubGlobal('fetch', (input: RequestInfo, init?: RequestInit) => {
    const url = String(input)
    const json = (status: number, payload: unknown) =>
      Promise.resolve(new Response(JSON.stringify(payload), { status }))
    if (url === '/api/publish/settings' && init?.method === 'PUT') {
      const body = JSON.parse(init.body as string) as unknown
      puts.push(body)
      return json(200, body)
    }
    if (url === '/api/publish/settings') {
      return json(200, { ...base, ...stored })
    }
    if (url === '/api/publish') {
      return json(202, { started: true })
    }
    throw new Error(`unexpected request: ${url}`)
  })
})

afterEach(() => {
  usePublishStore.getState().reset()
  vi.unstubAllGlobals()
})

describe('PublishDialog', () => {
  it('loads settings, saves them and starts a publish', async () => {
    const user = userEvent.setup()
    render(
      <>
        <PublishButton />
        <PublishDialog />
      </>,
    )
    await user.click(screen.getByRole('button', { name: 'Publish' }))

    const remote = await screen.findByLabelText('Remote')
    await waitFor(() => expect(remote).toHaveValue('git@example.com:a/b.git'))

    const branch = screen.getByLabelText('Branch')
    await user.clear(branch)
    await user.type(branch, 'pages')

    const buttons = screen.getAllByRole('button', { name: 'Publish' })
    await user.click(buttons.at(-1)!)

    await waitFor(() => expect(usePublishStore.getState().status).toBe('running'))
    expect(puts).toEqual([{ ...base, branch: 'pages' }])
  })

  it('shows rsync fields, warns before enabling delete and labels dry runs', async () => {
    stored = { target: 'rsync', rsyncDest: 'me@example.com:/srv/site' }
    const user = userEvent.setup()
    render(
      <>
        <PublishButton />
        <PublishDialog />
      </>,
    )
    await user.click(screen.getByRole('button', { name: 'Publish' }))

    const dest = await screen.findByLabelText('Destination')
    await waitFor(() => expect(dest).toHaveValue('me@example.com:/srv/site'))
    expect(screen.queryByLabelText('Remote')).toBeNull()

    const del = screen.getByLabelText(/Delete files on the destination/)
    await user.click(del)
    await user.click(screen.getByRole('button', { name: 'Keep it off' }))
    expect(del).not.toBeChecked()
    await user.click(del)
    expect(screen.getByText(/cannot be undone/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Enable delete' }))
    expect(del).toBeChecked()

    await user.click(screen.getByLabelText(/Dry run/))
    await user.type(screen.getByLabelText('Port'), '2222')
    await user.click(screen.getByRole('button', { name: 'Publish (dry run)' }))

    await waitFor(() => expect(usePublishStore.getState().status).toBe('running'))
    expect(puts).toEqual([
      {
        ...base,
        target: 'rsync',
        rsyncDest: 'me@example.com:/srv/site',
        rsyncPort: 2222,
        rsyncDelete: true,
        rsyncDryRun: true,
      },
    ])
  })
})
