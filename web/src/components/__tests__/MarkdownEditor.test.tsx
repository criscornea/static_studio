import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

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
})
