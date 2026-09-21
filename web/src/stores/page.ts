import { defineStore } from "pinia";
import { useProjectStore } from "./project";
import { asApiError, type ApiError, type Page } from "@/api/types";
import { api } from "@/api/client";
import { ref, watch } from "vue";

export const usePageStore = defineStore('page', () => {
  const project = useProjectStore()

  // The path most recently requested, set before the response arrives.
  const path = ref<string | null>(null)
  const page = ref<Page | null>(null)
  const loading = ref(false)
  const error = ref<ApiError | null>(null)

  // Incremented on every load. A response is only applied if no newer
  // load has started since, so fast clicking can't show a stale page.
  let latest = 0

  async function load(next: string): Promise<void> {
    const request = ++latest
    path.value = next
    loading.value = true
    error.value = null

    try {
      const loaded = await api.page(next)
      if (request !== latest) {
        return
      }
      page.value = loaded
    } catch (err) {
      if (request !== latest) {
        return
      }
      page.value = null
      error.value = asApiError(err)
    } finally {
      if (request === latest) {
        loading.value = false
      }
    }
  }

  function clear(): void {
    latest++
    path.value = null
    page.value = null
    error.value = null
    loading.value = false
  }

  // A different project, or none, means the open page no longer applies.
  watch(() => project.project?.id, clear)

  return { path, page, loading, error, load, clear }
})
