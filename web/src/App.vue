<script setup lang="ts">
import {computed, onMounted} from 'vue'
import {useProjectStore} from '@/stores/project'
import {usePageStore} from '@/stores/page'
import ProjectPicker from '@/components/ProjectPicker.vue'
import ContentTree from '@/components/ContentTree.vue'
import PageView from '@/components/PageView.vue'

const project = useProjectStore()
const pages = usePageStore()

const selected = computed<string | null>({
  get: () => pages.path,
  set: (path) => {
    if (path !== null) {
      pages.load(path)
    }
  }
})

// The backend holds the open project, so a page reload reconnects to it.
onMounted(() => project.restore())
</script>

<template>
  <ProjectPicker v-if="!project.isOpen" />

  <div v-else class="shell">
    <header>
      <span class="root">{{ project.project?.root }}</span>
      <button @click="project.close()">Close</button>
    </header>

    <div class="body">
      <nav class="sidebar">
        <ul v-if="project.tree?.children?.length">
          <ContentTree
            v-for="child in project.tree.children"
            :key="child.path"
            v-model:selected="selected"
            :node="child"
          />
        </ul>
        <p v-else class="empty">No content files yet.</p>
      </nav>

      <main class="content">
        <PageView />
      </main>
    </div>
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

header {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #e5e5e5;
}

.root {
  flex: 1;
  font-family: ui-monospace, monospace;
  font-size: 0.8125rem;
  color: #666;
}

.body {
  display: flex;
  flex: 1;
  min-height: 0;
}

.sidebar {
  width: 16rem;
  padding: 0.5rem;
  border-right: 1px solid #e5e5e5;
  overflow-y: auto;
}

.sidebar ul {
  list-style: none;
  margin: 0;
  padding: 0;
}

.content {
  flex: 1;
  padding: 2rem;
  overflow-y: auto;
}

.empty {
  color: #999;
  font-size: 0.875rem;
}
</style>
