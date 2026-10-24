import { create } from 'zustand'

import { api } from '@/api/client'
import { asApiError } from '@/api/types'

// Fast while the server starts, slow while it runs to notice a crash.
const startPollMs = 500
const runPollMs = 3000

interface DevServerState {
  running: boolean
  ready: boolean
  url: string | null
  // A start or stop request is in flight.
  busy: boolean
  error: string | null
  // Last output lines of a server that exited on its own.
  output: string[]

  refresh: () => Promise<void>
  start: () => Promise<void>
  stop: () => Promise<void>
  reset: () => void
}

let timer: ReturnType<typeof setTimeout> | null = null

function stopPolling() {
  if (timer !== null) {
    clearTimeout(timer)
    timer = null
  }
}

const idle = { running: false, ready: false, url: null, busy: false, error: null, output: [] as string[] }

export const useDevServerStore = create<DevServerState>()((set, get) => {
  // Poll while the server runs, so the UI learns when it is ready or died.
  function schedulePoll() {
    stopPolling()
    const delay = get().ready ? runPollMs : startPollMs
    timer = setTimeout(() => {
      timer = null
      void get().refresh()
    }, delay)
  }

  // Bumped by stop/reset so a late response cannot revive a stopped server.
  let epoch = 0

  return {
    ...idle,

    async refresh() {
      const seen = epoch
      try {
        const st = await api.devServer()
        if (seen !== epoch) {
          return
        }
        const output = st.output ?? []
        set({
          running: st.running,
          ready: st.ready,
          url: st.url ?? null,
          output,
          error: !st.running && output.length > 0 ? 'The preview server stopped unexpectedly.' : null,
        })
        if (st.running) {
          schedulePoll()
        } else {
          stopPolling()
        }
      } catch (err) {
        if (seen === epoch) {
          stopPolling()
          set({ error: asApiError(err).message })
        }
      }
    },

    async start() {
      if (get().busy || get().running) {
        return
      }
      set({ busy: true, error: null, output: [] })

      try {
        const st = await api.startDevServer()
        set({ running: st.running, ready: st.ready, url: st.url ?? null, busy: false })
        schedulePoll()
      } catch (err) {
        set({ busy: false, error: asApiError(err).message })
      }
    },

    async stop() {
      stopPolling()
      epoch++
      set({ busy: true, error: null, output: [] })

      try {
        await api.stopDevServer()
        set({ ...idle })
      } catch (err) {
        set({ busy: false, error: asApiError(err).message })
      }
    },

    reset() {
      stopPolling()
      epoch++
      set({ ...idle })
    },
  }
})
