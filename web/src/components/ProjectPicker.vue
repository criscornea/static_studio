<script setup lang="ts">
import {ref} from 'vue'
import {useProjectStore} from '@/stores/project'

const store = useProjectStore()
const path = ref('')

async function submit() {
const trimmed = path.value.trim()
if (trimmed === '') {
return
}
await store.open(trimmed)
}
</script>

<template>
  <section class="picker">
    <h1>static_studio</h1>
    <p class="lead">Open the folder that contains your website.</p>

    <div class="row">
      <input
        v-model="path"
        type="text"
        placeholder="/path/to/your/site"
        spellcheck="false"
        autocomplete="off"
        :disabled="store.loading"
        @keyup.enter="submit"
      />
      <button :disabled="store.loading || path.trim() === ''" @click="submit">
        {{ store.loading ? 'Opening…' : 'Open' }}
      </button>
    </div>

    <p v-if="store.error" class="error" role="alert">
      {{ store.error.message }}
    </p>

    <p class="hint">
      This is the folder with your <code>hugo.toml</code> in it.
    </p>
  </section>
</template>

<style scoped>
.picker {
  max-width: 34rem;
  margin: 4rem auto;
  padding: 0 1.5rem;
}

h1 {
  font-size: 1.5rem;
  margin: 0 0 0.25rem;
}

.lead {
  margin: 0 0 1.5rem;
  color: #666;
}

.row {
  display: flex;
  gap: 0.5rem;
}

input {
  flex: 1;
  padding: 0.6rem 0.75rem;
  font: inherit;
  font-family: ui-monospace, monospace;
  border: 1px solid #ccc;
  border-radius: 4px;
}

input:disabled {
  background: #f5f5f5;
}

button {
  padding: 0.6rem 1.25rem;
  font: inherit;
  cursor: pointer;
  border: 1px solid #ccc;
  border-radius: 4px;
  background: #fff;
}

button:disabled {
  cursor: default;
  opacity: 0.5;
}

.error {
  margin-top: 1rem;
  padding: 0.75rem;
  border-radius: 4px;
  background: #fdf0f0;
  color: #a33;
}

.hint {
  margin-top: 2rem;
  font-size: 0.875rem;
  color: #888;
}
</style>
