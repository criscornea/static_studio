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

// Node 25+ ships its own localStorage global, which is undefined without
// --localstorage-file and hides jsdom's. Fall back to an in-memory Storage.
if (typeof globalThis.localStorage === 'undefined') {
  const data = new Map<string, string>()
  const storage: Storage = {
    get length() {
      return data.size
    },
    clear: () => data.clear(),
    getItem: (key) => data.get(key) ?? null,
    key: (index) => [...data.keys()][index] ?? null,
    removeItem: (key) => void data.delete(key),
    setItem: (key, value) => void data.set(key, String(value)),
  }
  Object.defineProperty(globalThis, 'localStorage', {
    value: storage,
    configurable: true,
  })
}
