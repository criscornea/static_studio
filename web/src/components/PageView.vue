<script setup lang="ts">
import { computed } from 'vue'
import {usePageStore} from '@/stores/page'

const pages = usePageStore()

const dateFormat = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short',
})

const date = computed(() => {
  const raw = pages.page?.fields.date
  if (!raw) {
  return null
  }
  const parsed = new Date(raw)
  return Number.isNaN(parsed.getTime()) ? raw : dateFormat.format(parsed)
})

const formatLabel = computed(() => {
  switch (pages.page?.format) {
    case 'yaml':
      return 'YAML'
    case 'toml':
      return 'TOML'
    default:
      return 'no frontmatter'
  }
})
</script>

<template>
  <p v-if="pages.path === null" class="muted">Select a page.</p>

  <p v-else-if="pages.error" class="error" role="alert">
    {{ pages.error.message }}
  </p>

  <p v-else-if="pages.page === null" class="muted">Loading…</p>

  <article v-else class="page" :class="{ stale: pages.loading }">
    <header>
      <h1>{{ pages.page.fields.title || 'Untitled' }}</h1>

      <dl class="meta">
        <div v-if="date">
          <dt>Date</dt>
          <dd>{{ date }}</dd>
        </div>
        <div>
          <dt>Status</dt>
          <dd>
            <span class="badge" :class="{ draft: pages.page.fields.draft }">
              {{ pages.page.fields.draft ? 'Draft' : 'Published' }}
            </span>
          </dd>
        </div>
        <div v-if="pages.page.fields.tags?.length">
          <dt>Tags</dt>
          <dd class="tags">
            <span v-for="tag in pages.page.fields.tags" :key="tag" class="tag">{{ tag }}</span>
          </dd>
        </div>
        <div>
          <dt>Format</dt>
          <dd>{{ formatLabel }}</dd>
        </div>
      </dl>

      <p class="path">{{ pages.page.path }}</p>
    </header>

    <pre class="body">{{ pages.page.body }}</pre>
  </article>
</template>

<style scoped>
.muted {
  color: #999;
  font-size: 0.875rem;
}

.error {
  padding: 0.75rem;
  border-radius: 4px;
  background: #fdf0f0;
  color: #a33;
}

.page.stale {
  opacity: 0.5;
}

h1 {
  margin: 0 0 1rem;
  font-size: 1.5rem;
}

.meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 2rem;
  margin: 0 0 0.75rem;
}

.meta dt {
  font-size: 0.75rem;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.meta dd {
  margin: 0.125rem 0 0;
  font-size: 0.875rem;
}

.badge {
  padding: 0.1rem 0.5rem;
  border-radius: 999px;
  background: #e8f5e9;
  color: #2e7d32;
  font-size: 0.75rem;
}

.badge.draft {
  background: #fff3e0;
  color: #b25e00;
}

.tags {
  display: flex;
  gap: 0.25rem;
}

.tag {
  padding: 0.1rem 0.5rem;
  border-radius: 3px;
  background: #f0f0f0;
  font-size: 0.75rem;
}

.path {
  margin: 0;
  font-family: ui-monospace, monospace;
  font-size: 0.75rem;
  color: #aaa;
}

.body {
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid #eee;
  white-space: pre-wrap;
  font-family: ui-monospace, monospace;
  font-size: 0.875rem;
  line-height: 1.6;
}
</style>
