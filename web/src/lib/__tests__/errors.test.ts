import { describe, expect, it } from 'vitest'
import { readableMessage } from '../errors'

describe('readableMessage', () => {
  it('maps known codes', () => {
    expect(readableMessage('page_conflict', 'hash mismatch')).toMatch(/changed on disk/)
  })
  it('keeps server text for other codes', () => {
    expect(readableMessage('invalid_name', 'Name "x" is bad')).toBe('Name "x" is bad')
  })
})
