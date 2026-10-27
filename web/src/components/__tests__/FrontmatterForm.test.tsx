import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { PageFields } from '@/api/types'
import { FrontmatterForm } from '../FrontmatterForm'

const fields: PageFields = { title: 'Hello', draft: false, tags: ['a'], date: '2026-01-01T00:00:00Z' }

describe('FrontmatterForm', () => {
  it('shows title, date, draft and a Tags button, without a toggle', () => {
    render(<FrontmatterForm fields={fields} onChange={() => {}} />)
    expect(screen.getByRole('heading', { name: 'Hello' })).toBeInTheDocument()
    expect(screen.getByText('2026-01-01T00:00:00Z')).toBeInTheDocument()
    expect(screen.getByLabelText('Draft')).not.toBeChecked()
    expect(screen.getByRole('button', { name: 'Tags' })).toBeInTheDocument()
    expect(screen.queryByText('Static Studio YAML')).not.toBeInTheDocument()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Properties/ })).not.toBeInTheDocument()
  })

  it('edits the title via heading click, committing on Enter', async () => {
    const onChange = vi.fn<(f: PageFields) => void>()
    render(<FrontmatterForm fields={fields} onChange={onChange} />)
    await userEvent.click(screen.getByRole('heading', { name: 'Hello' }))
    await userEvent.type(screen.getByLabelText('Title'), ' world{Enter}')
    expect(onChange).toHaveBeenCalledWith({ ...fields, title: 'Hello world' })
    expect(screen.queryByLabelText('Title')).not.toBeInTheDocument()
  })

  it('edits the title via the pencil, committing on blur', async () => {
    const onChange = vi.fn<(f: PageFields) => void>()
    render(<FrontmatterForm fields={fields} onChange={onChange} />)
    await userEvent.click(screen.getByRole('button', { name: 'Edit title' }))
    await userEvent.type(screen.getByLabelText('Title'), '!')
    await userEvent.tab()
    expect(onChange).toHaveBeenCalledWith({ ...fields, title: 'Hello!' })
  })

  it('cancels title edit on Escape', async () => {
    const onChange = vi.fn<(f: PageFields) => void>()
    render(<FrontmatterForm fields={fields} onChange={onChange} />)
    await userEvent.click(screen.getByRole('button', { name: 'Edit title' }))
    await userEvent.type(screen.getByLabelText('Title'), 'x{Escape}')
    expect(onChange).not.toHaveBeenCalled()
    expect(screen.getByRole('heading', { name: 'Hello' })).toBeInTheDocument()
  })

  it('edits and clears the date, and toggles draft', async () => {
    const onChange = vi.fn<(f: PageFields) => void>()
    render(<FrontmatterForm fields={fields} onChange={onChange} />)
    await userEvent.click(screen.getByLabelText('Draft'))
    expect(onChange).toHaveBeenLastCalledWith({ ...fields, draft: true })

    await userEvent.click(screen.getByRole('button', { name: 'Edit date' }))
    await userEvent.clear(screen.getByLabelText(/Date/))
    await userEvent.keyboard('{Enter}')
    expect(onChange).toHaveBeenLastCalledWith({ ...fields, date: undefined })
  })
})
