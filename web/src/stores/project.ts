import { computed, ref } from "vue"
import { defineStore } from "pinia"
import { api, setProjectID } from "@/api/client"
import { ApiError, type ContentNode, type Project } from "@/api/types"

export const useProjectStore = defineStore('project', () => {
  const project = ref<Project | null>(null)
  const tree = ref<ContentNode | null>(null)
  const loading = ref(false)
  const error = ref<ApiError | null>(null)

  const isOpen = computed(() => project.value !== null)

  // Opens a project folder and loads its content tree.
  async function open(path: string): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      const opened = await api.openProject(path)
      project.value = opened
      setProjectID(opened.id)
      await loadTree()
      return true
    } catch (err) {
      reset()
      error.value = asApiError(err)
      return false
    } finally {
      loading.value = false
    }
  }

  // Restores the project the backend already has open, if any.
  async function restore(): Promise<void> {
    loading.value = true
    try {
      const current = await api.currentProject()
      project.value = current
      setProjectID(current.id)
      await loadTree()
    } catch (err) {
      // 'no_project' is the normal state on a fresh start, not a failure
      const apiErr = asApiError(err)
      reset()
      if (apiErr.code !== 'no_project') {
        error.value = apiErr
      }
    } finally {
      loading.value = false
    }
  }

  async function close(): Promise<void> {
    try {
      await api.closeProject()
    } finally {
      reset()
    }
  }

  // Reloads the content tree of the open project
  async function loadTree(): Promise<void> {
    if (project.value === null) {
      return
    }
    const { root } = await api.contentTree()
    tree.value = root
  }

  function reset(): void {
    project.value = null
    tree.value = null
    error.value = null
    setProjectID(null)
  }

  return { project, tree, loading, error, isOpen, open, restore, close, loadTree, reset }
})

function asApiError(err: unknown): ApiError {
  if (err instanceof ApiError) {
    return err
  }
  return new ApiError('unknonw', 'Something went wrong. Please try again.', 0)
}
