<script setup>
import { onMounted, ref } from 'vue'
import { checkHealth } from '../api/health'

const status = ref('memeriksa...')

onMounted(async () => {
  try {
    const data = await checkHealth()
    status.value = data.status
  } catch (err) {
    status.value = `gagal: ${err.message}`
  }
})
</script>

<template>
  <main class="flex min-h-screen flex-col items-center justify-center gap-4 bg-slate-950 text-slate-100">
    <h1 class="text-2xl font-semibold">SMEditor</h1>
    <p class="text-slate-400">Status backend: <span class="font-mono text-emerald-400">{{ status }}</span></p>
  </main>
</template>
