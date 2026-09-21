<script setup lang="ts">
import {ref} from 'vue'
import type {ContentNode} from '@/api/types'

defineProps<{
  node: ContentNode
  // Nesting level, used for indentation. The root renders its children at 0.
  depth?: number
}>()

const selected = defineModel<string | null>('selected')

// Each folder instance tracks its own state. Folders start expanded.
const expanded = ref(true)
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
    <button
      type="button"
      class="label"
      :aria-expanded="expanded"
      :style="{ paddingLeft: `${(depth ?? 0) * 0.75 + 0.5}rem` }"
      @click="expanded = !expanded"
    >
      <span class="chevron" :class="{ open: expanded }">›</span>
      {{ node.name }}
    </button>
    <ul v-show="expanded">
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
  display: flex;
  align-items: center;
  gap: 0.25rem;
  width: 100%;
  padding-block: 0.25rem;
  border: 0;
  background: none;
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  color: #888;
  text-align: left;
  cursor: pointer;
}

.chevron {
  display: inline-block;
  width: 0.75rem;
  transition: transform 0.1s;
}

.chevron.open {
  transform: rotate(90deg);
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
