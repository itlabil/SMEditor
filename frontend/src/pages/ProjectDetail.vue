<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { getProject, retryDownload } from '../api/projects'
import { useNotify } from '../composables/useNotify'
import { useSSE } from '../composables/useSSE'

const route = useRoute()
const { success, error } = useNotify()

const project = ref(null)
const loading = ref(true)
const retrying = ref(false)
const job = ref(null)

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

async function retry() {
  retrying.value = true
  try {
    project.value = await retryDownload(route.params.id)
    success('Unduh diulang')
  } catch (err) {
    error(err.message)
  } finally {
    retrying.value = false
  }
}

function onJobEvent(ev) {
  if (ev.type === 'progress') {
    job.value = { type: ev.job_type, progress: ev.progress, message: ev.message }
    return
  }
  // done, failed, canceled: reload from the API instead of guessing the
  // new project status client-side.
  job.value = null
  load()
}

const sse = useSSE(route.params.id, onJobEvent)

onMounted(load)
onUnmounted(() => sse.close())
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
        <template v-if="project.video_title">
          <dt class="text-slate-400">Judul video</dt>
          <dd>{{ project.video_title }}</dd>
          <dt class="text-slate-400">Resolusi</dt>
          <dd>{{ project.width }}x{{ project.height }} · {{ project.fps.toFixed(0) }}fps · {{ project.video_codec }}</dd>
        </template>
      </dl>

      <div v-if="job" class="flex flex-col gap-1">
        <div class="flex justify-between text-sm text-slate-400">
          <span>{{ job.type }}</span>
          <span>{{ Math.round(job.progress) }}%</span>
        </div>
        <div class="h-2 w-full overflow-hidden rounded bg-slate-800">
          <div class="h-full bg-emerald-500 transition-all" :style="{ width: job.progress + '%' }"></div>
        </div>
        <p v-if="job.message" class="text-xs text-slate-500">{{ job.message }}</p>
      </div>

      <div v-if="project.status === 'gagal_unduh'" class="flex flex-col gap-2 rounded border border-rose-800 bg-rose-950/40 p-3">
        <p class="text-sm text-rose-300">{{ project.error_message || 'Unduh gagal.' }}</p>
        <button
          :disabled="retrying"
          class="self-start rounded bg-emerald-600 px-3 py-1.5 text-sm font-medium disabled:opacity-50"
          @click="retry"
        >
          {{ retrying ? 'Mengulang...' : 'Ulangi unduh' }}
        </button>
      </div>
    </template>
  </main>
</template>
