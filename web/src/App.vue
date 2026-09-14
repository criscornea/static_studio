<script setup lang="ts">
import {ref, onMounted} from 'vue'

const status = ref<string>('checking...')

onMounted(async () => {
try {
  const res = await fetch('api/health')
  if (!res.ok) throw new Error(`HTTP ${res.status}`)

  const body = (await res.json()) as { status: string }
  status.value = body.status
} catch (err) {
  status.value = `unreachable: ${err instanceof Error ? err.message : String(err)}`
}
})
</script>

<template>
  <main>
    <h1>Static Studio</h1>
    <p>backend: {{ status }}</p>
  </main>
</template>

<style scoped></style>
