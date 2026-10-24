import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { PreviewButton, PreviewFrame } from '../DevServer'
import { useDevServerStore } from '@/stores/devserver'

beforeEach(() => {
  useDevServerStore.getState().reset()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('PreviewButton', () => {
  it('is disabled until the server is ready', () => {
    render(<PreviewButton open={false} onToggle={() => {}} />)
    expect(screen.getByRole('button', { name: 'Preview' })).toBeDisabled()
  })
})

describe('PreviewFrame', () => {
  it('loads the page from the dev server origin', async () => {
    useDevServerStore.setState({ running: true, ready: true, url: 'http://127.0.0.1:4000' })
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(
          new Response(
            JSON.stringify({
              running: true,
              ready: true,
              url: 'http://127.0.0.1:4000',
              pageUrl: 'http://127.0.0.1:4000/posts/a/',
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          ),
        ),
      ),
    )

    render(<PreviewFrame path="content/posts/a.md" />)
    const frame = await screen.findByTitle('Page preview')
    expect(frame).toHaveAttribute('src', 'http://127.0.0.1:4000/posts/a/')
  })
})
