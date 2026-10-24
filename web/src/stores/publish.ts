import { create } from 'zustand'

import { api } from '@/api/client'
import { asApiError, type BuildEvent } from '@/api/types'
import { maxLines } from '@/stores/build'

export type PublishStatus = 'idle' | 'running' | 'succeeded' | 'failed'

interface PublishState {
  status: PublishStatus
  lines: string[]
  error: string | null

  start: () => Promise<void>
  cancel: () => Promise<void>
  attach: () => void
  reset: () => void
}

let source: EventSource | null = null

function closeSource() {
  source?.close()
  source = null
}

export const usePublishStore = create<PublishState>()((set, get) => {
  function listen() {
    closeSource()
    const es = new EventSource(api.publishStreamURL())
    source = es

    es.onmessage = (msg: MessageEvent<string>) => {
      let ev: BuildEvent
      try {
        ev = JSON.parse(msg.data) as BuildEvent
      } catch {
        return
      }

      if (ev.kind === 'line') {
        if (get().status === 'idle') {
          set({ status: 'running', lines: [], error: null })
        }
        const lines = [...get().lines, ev.line ?? '']
        set({ lines: lines.length > maxLines ? lines.slice(-maxLines) : lines })
        return
      }

      set({ status: ev.ok ? 'succeeded' : 'failed', error: ev.ok ? null : (ev.error ?? null) })
      closeSource()
    }

    es.onerror = () => {
      if (source === es && get().status === 'running') {
        set({ status: 'failed', error: 'Lost the connection to the publish.' })
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
        await api.startPublish()
      } catch (err) {
        set({ status: 'failed', error: asApiError(err).message })
        return
      }
      listen()
    },

    async cancel() {
      try {
        await api.cancelPublish()
      } catch (err) {
        set({ error: asApiError(err).message })
      }
    },

    // Picks up a publish that is already running, for example after the
    // dialog was closed. The stream ends at once when there is nothing to show.
    attach() {
      if (get().status !== 'idle' || source) {
        return
      }
      listen()
    },

    reset() {
      closeSource()
      set({ status: 'idle', lines: [], error: null })
    },
  }
})
