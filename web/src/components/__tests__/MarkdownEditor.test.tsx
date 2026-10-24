import { act, render, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { imageMarkdown } from '@/lib/markdown'
import { useEditorModeStore } from '@/stores/editorMode'
import { MarkdownEditor } from '../MarkdownEditor'

describe('MarkdownEditor', () => {
  it('keeps CRLF line endings and does not report a change on open', () => {
    const onChange = vi.fn<(text: string) => void>()
    const { container } = render(
      <MarkdownEditor value={'a\r\nb\r\n'} onChange={onChange} onSave={() => {}} />,
    )
    expect(onChange).not.toHaveBeenCalled()
    expect(container.querySelectorAll('.cm-line')).toHaveLength(3)
  })

  it('builds an image link from the upload path', () => {
    expect(imageMarkdown('/images/cat.png', 'My [cat].png')).toBe('![My cat](/images/cat.png)')
  })

  it('inserts an image link on drop and reports upload errors', async () => {
    const onChange = vi.fn<(text: string) => void>()
    const onUploadError = vi.fn<(err: unknown) => void>()
    const good = new File(['x'], 'cat.png', { type: 'image/png' })
    const bad = new File(['x'], 'dog.png', { type: 'image/png' })
    const onUploadImage = vi.fn<(f: File) => Promise<string>>(async (f) => {
      if (f === bad) throw new Error('nope')
      return '/images/cat.png'
    })
    const { container } = render(
      <MarkdownEditor
        value={'hello'}
        onChange={onChange}
        onSave={() => {}}
        onUploadImage={onUploadImage}
        onUploadError={onUploadError}
      />,
    )
    const content = container.querySelector('.cm-content')!
    const drop = new Event('drop', { bubbles: true, cancelable: true })
    Object.assign(drop, { dataTransfer: { files: [good, bad], items: [] }, clientX: 0, clientY: 0 })
    content.dispatchEvent(drop)

    await waitFor(() => expect(onUploadError).toHaveBeenCalledTimes(1))
    expect(onUploadError.mock.calls[0]?.[0]).toEqual(new Error('nope'))
    const text = onChange.mock.calls.at(-1)?.[0] ?? ''
    expect(text).toContain('![cat](/images/cat.png)')
    expect(text).toContain('hello')
  })

  it('toggles live preview without changing the text or reporting a change', () => {
    const onChange = vi.fn<(text: string) => void>()
    useEditorModeStore.getState().setMode('live')
    const { container } = render(
      <MarkdownEditor value={'# Title\n\n**b**'} onChange={onChange} onSave={() => {}} />,
    )
    const text = () =>
      Array.from(container.querySelectorAll('.cm-line'))
        .map((l) => l.textContent)
        .join('\n')
    act(() => useEditorModeStore.getState().setMode('source'))
    expect(text()).toBe('# Title\n\n**b**')
    act(() => useEditorModeStore.getState().setMode('live'))
    expect(onChange).not.toHaveBeenCalled()
    useEditorModeStore.getState().setMode('live')
  })
})
