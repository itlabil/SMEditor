<script setup>
import { onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { getProject } from '../api/projects'
import { useNotify } from '../composables/useNotify'

const route = useRoute()
const { error } = useNotify()

const project = ref(null)
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    project.value = await getProject(route.params.id)
  } catch (err) {
    error(err.message)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-2xl flex-col gap-4 bg-slate-950 p-8 text-slate-100">
    <RouterLink to="/" class="text-sm text-emerald-400 underline">&larr; Kembali ke daftar project</RouterLink>

    <p v-if="loading" class="text-slate-400">Memuat...</p>
    <template v-else-if="project">
      <h1 class="text-2xl font-semibold">{{ project.name }}</h1>
      <dl class="grid grid-cols-2 gap-2 text-sm">
        <dt class="text-slate-400">Game</dt>
        <dd>{{ project.game_code }}</dd>
        <dt class="text-slate-400">Status</dt>
        <dd>{{ project.status }}</dd>
        <dt class="text-slate-400">URL YouTube</dt>
        <dd class="truncate">{{ project.youtube_url }}</dd>
        <dt class="text-slate-400">Tim</dt>
        <dd>{{ project.team_a || '-' }} vs {{ project.team_b || '-' }}</dd>
      </dl>
    </template>
  </main>
</template>
