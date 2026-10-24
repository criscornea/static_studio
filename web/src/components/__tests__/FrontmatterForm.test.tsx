import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useLayoutStore } from '@/stores/layout'
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

describe('FrontmatterForm panel', () => {
  afterEach(() => {
    useLayoutStore.setState({ propertiesOpen: true })
  })

  it('collapses and expands, keeping the fields mounted', async () => {
    render(<FrontmatterForm fields={fields} onChange={() => {}} />)
    const toggle = screen.getByRole('button', { name: /Properties/ })
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    await userEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(useLayoutStore.getState().propertiesOpen).toBe(false)
    expect(screen.getByLabelText('Title', { selector: 'input' })).not.toBeVisible()
    await userEvent.click(toggle)
    expect(screen.getByLabelText('Title')).toBeVisible()
  })

  it('shows a summary while collapsed', () => {
    useLayoutStore.setState({ propertiesOpen: false })
    render(<FrontmatterForm fields={{ ...fields, draft: true }} format="yaml" onChange={() => {}} />)
    expect(screen.getByRole('button', { name: /Hello · draft · YAML/ })).toBeInTheDocument()
  })
})
