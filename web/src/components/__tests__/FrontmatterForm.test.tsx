import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { PageFields } from '@/api/types'
import { FrontmatterForm } from '../FrontmatterForm'

const fields: PageFields = { title: 'Hello', draft: false, tags: ['a'], date: '2026-01-01T00:00:00Z' }

describe('FrontmatterForm', () => {
  it('shows title, date, draft and a Tags button, without a toggle', () => {
    render(<FrontmatterForm fields={fields} onChange={() => {}} />)
    expect(screen.getByRole('button', { name: 'Edit title' })).toHaveTextContent('Hello')
    expect(screen.getByText('2026-01-01T00:00:00Z')).toBeInTheDocument()
    expect(screen.getByLabelText('Draft')).not.toBeChecked()
    expect(screen.getByRole('button', { name: 'Tags' })).toBeInTheDocument()
    expect(screen.queryByText('Static Studio YAML')).not.toBeInTheDocument()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Properties/ })).not.toBeInTheDocument()
  })

  it('edits the title via text click, committing on Enter', async () => {
    const onChange = vi.fn<(f: PageFields) => void>()
    render(<FrontmatterForm fields={fields} onChange={onChange} />)
    await userEvent.click(screen.getByRole('button', { name: 'Edit title' }))
    await userEvent.type(screen.getByLabelText('Title'), ' world{Enter}')
    expect(onChange).toHaveBeenCalledWith({ ...fields, title: 'Hello world' })
    expect(screen.queryByLabelText('Title')).not.toBeInTheDocument()
  })

  it('edits the title via keyboard Enter, committing on blur', async () => {
    const onChange = vi.fn<(f: PageFields) => void>()
    render(<FrontmatterForm fields={fields} onChange={onChange} />)
    screen.getByRole('button', { name: 'Edit title' }).focus()
    await userEvent.keyboard('{Enter}')
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
    expect(screen.getByRole('button', { name: 'Edit title' })).toBeInTheDocument()
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

  it('sets the date from a calendar day click, keeping time and zone', async () => {
    const onChange = vi.fn<(f: PageFields) => void>()
    const f = { ...fields, date: '2026-01-01T09:30:00+01:00' }
    render(<FrontmatterForm fields={f} onChange={onChange} />)
    await userEvent.click(screen.getByRole('button', { name: 'Edit date' }))
    await userEvent.click(screen.getByRole('button', { name: /January 15th, 2026/ }))
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith({ ...f, date: '2026-01-15T09:30:00+01:00' })
    expect(screen.queryByLabelText(/Date/)).not.toBeInTheDocument()
  })

  it('still accepts typing, and calendar follows the typed date', async () => {
    const onChange = vi.fn<(f: PageFields) => void>()
    render(<FrontmatterForm fields={fields} onChange={onChange} />)
    await userEvent.click(screen.getByRole('button', { name: 'Edit date' }))
    const input = screen.getByLabelText(/Date/)
    await userEvent.clear(input)
    await userEvent.type(input, '2027-03-05T10:00:00Z')
    expect(screen.getByText(/March 2027/)).toBeInTheDocument()
    await userEvent.keyboard('{Enter}')
    expect(onChange).toHaveBeenCalledWith({ ...fields, date: '2027-03-05T10:00:00Z' })
  })

  it('does not blur-commit or cancel when pressing inside the calendar', async () => {
    const onChange = vi.fn<(f: PageFields) => void>()
    render(<FrontmatterForm fields={fields} onChange={onChange} />)
    await userEvent.click(screen.getByRole('button', { name: 'Edit date' }))
    const input = screen.getByLabelText(/Date/)
    await userEvent.type(input, 'x')
    await userEvent.click(screen.getByRole('button', { name: /Go to the Next Month/i }))
    expect(onChange).not.toHaveBeenCalled()
    expect(screen.getByLabelText(/Date/)).toHaveFocus()
  })
})
