import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { PageFields } from '@/api/types'
import { FrontmatterForm } from '../FrontmatterForm'

const fields: PageFields = { title: 'Hello', draft: false, tags: ['a'], date: '2026-01-01T00:00:00Z' }

describe('FrontmatterForm', () => {
  it('shows the fields', () => {
    render(<FrontmatterForm fields={fields} onChange={() => {}} />)
    expect(screen.getByLabelText('Title')).toHaveValue('Hello')
    expect(screen.getByLabelText(/Tags/)).toHaveValue('a')
    expect(screen.getByLabelText(/Date/)).toHaveValue('2026-01-01T00:00:00Z')
    expect(screen.getByLabelText('Draft')).not.toBeChecked()
  })

  it('reports edits', async () => {
    const onChange = vi.fn<(f: PageFields) => void>()
    render(<FrontmatterForm fields={fields} onChange={onChange} />)
    await userEvent.click(screen.getByLabelText('Draft'))
    expect(onChange).toHaveBeenLastCalledWith({ ...fields, draft: true })

    await userEvent.type(screen.getByLabelText(/Tags/), ', b')
    expect(onChange).toHaveBeenLastCalledWith({ ...fields, tags: ['a', 'b'] })

    await userEvent.clear(screen.getByLabelText(/Date/))
    expect(onChange).toHaveBeenLastCalledWith({ ...fields, date: undefined })
  })
})
