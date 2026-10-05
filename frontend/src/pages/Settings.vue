<script setup>
import { onMounted, ref } from 'vue'
import { getSettings, updateSettings, checkTools } from '../api/settings'
import { useNotify } from '../composables/useNotify'

const { success, error } = useNotify()

const form = ref({
  ytdlp_path: '',
  ffmpeg_path: '',
  ffprobe_path: '',
  whisper_path: '',
  whisper_model: '',
  whisper_device: 'auto',
})
const loading = ref(true)
const saving = ref(false)
const checking = ref(false)
const checkResults = ref([])

async function load() {
  loading.value = true
  try {
    form.value = await getSettings()
  } catch (err) {
    error(err.message)
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    form.value = await updateSettings(form.value)
    success('Pengaturan tersimpan')
  } catch (err) {
    error(err.message)
  } finally {
    saving.value = false
  }
}

async function runCheck() {
  checking.value = true
  try {
    checkResults.value = await checkTools()
  } catch (err) {
    error(err.message)
  } finally {
    checking.value = false
  }
}

onMounted(load)
</script>

<template>
  <main class="mx-auto flex min-h-screen max-w-2xl flex-col gap-6 bg-slate-950 p-8 text-slate-100">
    <h1 class="text-2xl font-semibold">Pengaturan</h1>

    <form v-if="!loading" class="flex flex-col gap-4" @submit.prevent="save">
      <label class="flex flex-col gap-1">
        <span class="text-sm text-slate-400">Path yt-dlp</span>
        <input v-model="form.ytdlp_path" class="rounded bg-slate-800 px-3 py-2" />
      </label>
      <label class="flex flex-col gap-1">
        <span class="text-sm text-slate-400">Path ffmpeg</span>
        <input v-model="form.ffmpeg_path" class="rounded bg-slate-800 px-3 py-2" />
      </label>
      <label class="flex flex-col gap-1">
        <span class="text-sm text-slate-400">Path ffprobe</span>
        <input v-model="form.ffprobe_path" class="rounded bg-slate-800 px-3 py-2" />
      </label>
      <label class="flex flex-col gap-1">
        <span class="text-sm text-slate-400">Path Whisper</span>
        <input v-model="form.whisper_path" class="rounded bg-slate-800 px-3 py-2" />
      </label>
      <label class="flex flex-col gap-1">
        <span class="text-sm text-slate-400">Model Whisper</span>
        <input v-model="form.whisper_model" class="rounded bg-slate-800 px-3 py-2" />
      </label>
      <label class="flex flex-col gap-1">
        <span class="text-sm text-slate-400">Perangkat Whisper</span>
        <select v-model="form.whisper_device" class="rounded bg-slate-800 px-3 py-2">
          <option value="auto">Otomatis</option>
          <option value="cpu">CPU</option>
          <option value="gpu">GPU</option>
        </select>
      </label>

      <button
        type="submit"
        :disabled="saving"
        class="rounded bg-emerald-600 px-4 py-2 font-medium disabled:opacity-50"
      >
        {{ saving ? 'Menyimpan...' : 'Simpan' }}
      </button>
    </form>

    <section class="flex flex-col gap-3 border-t border-slate-800 pt-6">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-semibold">Pemeriksaan tool</h2>
        <button
          :disabled="checking"
          class="rounded bg-slate-800 px-4 py-2 font-medium disabled:opacity-50"
          @click="runCheck"
        >
          {{ checking ? 'Memeriksa...' : 'Periksa Tool' }}
        </button>
      </div>

      <ul v-if="checkResults.length" class="flex flex-col gap-2">
        <li
          v-for="r in checkResults"
          :key="r.tool"
          class="flex items-center justify-between rounded bg-slate-900 px-3 py-2"
        >
          <div>
            <p class="font-medium">{{ r.tool }}</p>
            <p class="font-mono text-xs text-slate-500">{{ r.path }}</p>
          </div>
          <span :class="r.found ? 'text-emerald-400' : 'text-rose-400'">
            {{ r.found ? `ditemukan (${r.version || 'versi tidak terbaca'})` : 'tidak ditemukan' }}
          </span>
        </li>
      </ul>
    </section>
  </main>
</template>
