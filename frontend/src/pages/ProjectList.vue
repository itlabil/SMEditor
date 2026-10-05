<script setup>
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { listProjects, createProject, deleteProject, thumbnailUrl } from '../api/projects'
import { useNotify } from '../composables/useNotify'
import { useConfirm } from '../composables/useConfirm'

const { success, error } = useNotify()
const { confirm } = useConfirm()

const projects = ref([])
const loading = ref(true)
const creating = ref(false)

const form = ref({ name: '', game_code: '', youtube_url: '', team_a: '', team_b: '', target_minutes: 0 })

async function load() {
  loading.value = true
  try {
    projects.value = await listProjects()
  } catch (err) {
    error(err.message)
  } finally {
    loading.value = false
  }
}

async function create() {
  creating.value = true
  try {
    await createProject(form.value)
    form.value = { name: '', game_code: '', youtube_url: '', team_a: '', team_b: '', target_minutes: 0 }
    success('Project dibuat')
    await load()
  } catch (err) {
    error(err.message)
  } finally {
    creating.value = false
  }
}

async function remove(p) {
  const ok = await confirm({
    title: 'Hapus project?',
    text: `"${p.name}" dan semua filenya akan dihapus permanen.`,
  })
  if (!ok) return
  try {
    await deleteProject(p.id)
    success('Project dihapus')
    await load()
  } catch (err) {
    error(err.message)
  }
}

function formatSize(bytes) {
  if (!bytes) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let n = bytes
  let i = 0
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i += 1
  }
  return `${n.toFixed(1)} ${units[i]}`
}

function formatDuration(seconds) {
  if (!seconds) return '-'
  const total = Math.round(seconds)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const pad = (v) => String(v).padStart(2, '0')
  return h > 0 ? `${pad(h)}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`
}

onMounted(load)
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 bg-slate-950 p-8 text-slate-100">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold">Project</h1>
      <RouterLink to="/settings" class="text-sm text-emerald-400 underline">Pengaturan</RouterLink>
    </div>

    <form class="flex flex-col gap-3 rounded border border-slate-800 p-4" @submit.prevent="create">
      <h2 class="text-lg font-semibold">Project baru</h2>
      <label class="flex flex-col gap-1">
        <span class="text-sm text-slate-400">Nama project</span>
        <input v-model="form.name" required class="rounded bg-slate-800 px-3 py-2" />
      </label>
      <label class="flex flex-col gap-1">
        <span class="text-sm text-slate-400">Kode game (contoh: mlbb, valorant, umum)</span>
        <input v-model="form.game_code" required class="rounded bg-slate-800 px-3 py-2" />
      </label>
      <label class="flex flex-col gap-1">
        <span class="text-sm text-slate-400">URL YouTube</span>
        <input v-model="form.youtube_url" required class="rounded bg-slate-800 px-3 py-2" />
      </label>
      <div class="grid grid-cols-2 gap-3">
        <label class="flex flex-col gap-1">
          <span class="text-sm text-slate-400">Tim A (opsional)</span>
          <input v-model="form.team_a" class="rounded bg-slate-800 px-3 py-2" />
        </label>
        <label class="flex flex-col gap-1">
          <span class="text-sm text-slate-400">Tim B (opsional)</span>
          <input v-model="form.team_b" class="rounded bg-slate-800 px-3 py-2" />
        </label>
      </div>
      <button
        type="submit"
        :disabled="creating"
        class="rounded bg-emerald-600 px-4 py-2 font-medium disabled:opacity-50"
      >
        {{ creating ? 'Membuat...' : 'Buat project' }}
      </button>
    </form>

    <section class="flex flex-col gap-3">
      <h2 class="text-lg font-semibold">Daftar project</h2>
      <p v-if="loading" class="text-slate-400">Memuat...</p>
      <p v-else-if="!projects.length" class="text-slate-400">Belum ada project.</p>
      <ul v-else class="flex flex-col gap-2">
        <li
          v-for="p in projects"
          :key="p.id"
          class="flex items-center gap-4 rounded bg-slate-900 px-3 py-2"
        >
          <img
            v-if="p.has_thumbnail"
            :src="thumbnailUrl(p.id)"
            alt=""
            class="h-12 w-20 flex-none rounded object-cover"
          />
          <div v-else class="h-12 w-20 flex-none rounded bg-slate-800"></div>

          <RouterLink :to="`/projects/${p.id}`" class="flex-1 min-w-0">
            <p class="truncate font-medium">{{ p.name }}</p>
            <p class="text-xs text-slate-400">
              {{ p.game_code }} · {{ p.status }} · {{ formatDuration(p.duration_sec) }} · {{ formatSize(p.size_bytes) }}
            </p>
          </RouterLink>

          <button class="rounded bg-rose-700 px-3 py-1 text-sm" @click="remove(p)">Hapus</button>
        </li>
      </ul>
    </section>
  </main>
</template>
