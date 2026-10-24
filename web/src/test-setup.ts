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

// jsdom has no scrollIntoView, which cmdk calls for the highlighted item.
Element.prototype.scrollIntoView ??= () => {}

// jsdom lacks pointer capture, which Radix Select calls on pointer down.
Element.prototype.hasPointerCapture ??= () => false
Element.prototype.setPointerCapture ??= () => {}
Element.prototype.releasePointerCapture ??= () => {}
