import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// Testing Library only unmounts automatically when vitest globals are on.
afterEach(cleanup)

// jsdom has no ResizeObserver, which Radix's ScrollArea and Select need.
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
}
