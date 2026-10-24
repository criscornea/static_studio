import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { usePublishStore } from '@/stores/publish'
import { PublishButton } from '../PublishDialog'

let puts: unknown[] = []

beforeEach(() => {
  puts = []
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
      return json(200, { remote: 'git@example.com:a/b.git', branch: 'gh-pages' })
    }
    if (url === '/api/publish') {
      return json(202, { started: true })
    }
    throw new Error(`unexpected request: ${url}`)
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('PublishDialog', () => {
  it('loads settings, saves them and starts a publish', async () => {
    const user = userEvent.setup()
    render(<PublishButton />)
    await user.click(screen.getByRole('button', { name: 'Publish' }))

    const remote = await screen.findByLabelText('Remote')
    await waitFor(() => expect(remote).toHaveValue('git@example.com:a/b.git'))

    const branch = screen.getByLabelText('Branch')
    await user.clear(branch)
    await user.type(branch, 'pages')

    const buttons = screen.getAllByRole('button', { name: 'Publish' })
    await user.click(buttons.at(-1)!)

    await waitFor(() => expect(usePublishStore.getState().status).toBe('running'))
    expect(puts).toEqual([{ remote: 'git@example.com:a/b.git', branch: 'pages' }])
  })
})
