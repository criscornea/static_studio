<script setup lang="ts">
import type {ContentNode} from '@/api/types'

defineProps<{
  node: ContentNode
  // Nesting level, used for indentation. The root renders its children at 0.
  depth?: number
}>()

const selected = defineModel<string | null>('selected')

function toggle(path: string, open: Set<string>) {
  if (open.has(path)) {
    open.delete(path)
    return
  }
  open.add(path)
}
</script>

<template>
  <li v-if="!node.isDir" class="file">
    <button
      type="button"
      :class="{ active: selected === node.path }"
      :style="{ paddingLeft: `${(depth ?? 0) * 0.75 + 0.5}rem` }"
      @click="selected = node.path"
    >
      {{ node.name }}
    </button>
  </li>

  <li v-else class="dir">
    <span class="label" :style="{ paddingLeft: `${(depth ?? 0) * 0.75 + 0.5}rem` }">
      {{ node.name }}
    </span>
    <ul>
      <ContentTree
        v-for="child in node.children"
        :key="child.path"
        v-model:selected="selected"
        :node="child"
        :depth="(depth ?? 0) + 1"
      />
    </ul>
  </li>
</template>

<style scoped>
ul {
  list-style: none;
  margin: 0;
  padding: 0;
}

.label {
  display: block;
  padding-block: 0.25rem;
  font-size: 0.8125rem;
  font-weight: 600;
  color: #888;
}

.file button {
  display: block;
  width: 100%;
  padding-block: 0.25rem;
  padding-right: 0.5rem;
  border: 0;
  background: none;
  font: inherit;
  font-size: 0.875rem;
  text-align: left;
  cursor: pointer;
  border-radius: 3px;
}

.file button:hover {
  background: #f0f0f0;
}

.file button.active {
  background: #e4ecf7;
  color: #1a4b8c;
}
</style>
