import { describe, expect, it } from 'vitest'
import { readableMessage } from '../errors'

describe('readableMessage', () => {
  it('maps known codes', () => {
    expect(readableMessage('page_conflict', 'hash mismatch')).toMatch(/changed on disk/)
  })
  it('keeps the server text for invalid_config so the file and reason show', () => {
    const msg = 'The site\'s configuration file hugo.toml could not be used: line 2: bad.'
    expect(readableMessage('invalid_config', msg)).toBe(msg)
  })
  it('keeps server text for other codes', () => {
    expect(readableMessage('invalid_name', 'Name "x" is bad')).toBe('Name "x" is bad')
  })
})
