import { create } from 'zustand'

import { api } from '@/api/client'
import { asApiError, type BuildEvent } from '@/api/types'

/** Older lines are dropped so a chatty build cannot grow the page forever. */
export const maxLines = 5000

export type BuildStatus = 'idle' | 'running' | 'succeeded' | 'failed'

interface BuildState {
  status: BuildStatus
  lines: string[]
  error: string | null

  start: () => Promise<void>
  cancel: () => Promise<void>
  reset: () => void
}

let source: EventSource | null = null

function closeSource() {
  source?.close()
  source = null
}

export const useBuildStore = create<BuildState>()((set, get) => {
  function listen() {
    closeSource()
    const es = new EventSource(api.buildStreamURL())
    source = es

    es.onmessage = (msg: MessageEvent<string>) => {
      let ev: BuildEvent
      try {
        ev = JSON.parse(msg.data) as BuildEvent
      } catch {
        return
      }

      if (ev.kind === 'line') {
        const lines = [...get().lines, ev.line ?? '']
        set({ lines: lines.length > maxLines ? lines.slice(-maxLines) : lines })
        return
      }

      set({ status: ev.ok ? 'succeeded' : 'failed', error: ev.ok ? null : (ev.error ?? null) })
      closeSource()
    }

    es.onerror = () => {
      // The server closes the stream after "done". Anything else is a drop.
      if (source === es && get().status === 'running') {
        set({ status: 'failed', error: 'Lost the connection to the build.' })
      }
      closeSource()
    }
  }

  return {
    status: 'idle',
    lines: [],
    error: null,

    async start() {
      if (get().status === 'running') {
        return
      }
      set({ status: 'running', lines: [], error: null })

      try {
        await api.startBuild()
      } catch (err) {
        set({ status: 'failed', error: asApiError(err).message })
        return
      }
      listen()
    },

    async cancel() {
      try {
        await api.cancelBuild()
      } catch (err) {
        set({ error: asApiError(err).message })
      }
    },

    reset() {
      closeSource()
      set({ status: 'idle', lines: [], error: null })
    },
  }
})
