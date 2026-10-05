<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { getProject, getPrompt, openFolder, retryDownload, retryTranscribe, transcriptUrl, videoUrl } from '../api/projects'
import { listGameModes } from '../api/promptBlocks'
import { saveHighlight, getHighlight, deleteHighlight, narasiUrl } from '../api/highlight'
import { cancelJob } from '../api/jobs'
import { useNotify } from '../composables/useNotify'
import { useConfirm } from '../composables/useConfirm'
import { useSSE } from '../composables/useSSE'
import { statusLabel, gameName, jobTypeLabel, categoryColor } from '../lib/labels'
import AppLayout from '../components/AppLayout.vue'

const route = useRoute()
const { success, error } = useNotify()
const { confirm } = useConfirm()

const project = ref(null)
const gameModes = ref([])
const loading = ref(true)
const retrying = ref(false)
const retryingTranscript = ref(false)
const cancellingJob = ref(false)
const transcriptLang = ref('auto')
const job = ref(null)
const promptText = ref('')
const openingFolder = ref(false)

const highlightBody = ref('')
const highlightErrors = ref([])
const savingHighlight = ref(false)
const deletingHighlight = ref(false)
const savedHighlight = ref(null)

const videoEl = ref(null)
const activeSegmentIndex = ref(-1)
let stopAtSeconds = null

const pastDownload = computed(() =>
  ['transcript', 'gagal_transcript', 'menunggu_highlight', 'siap_premiere'].includes(project.value?.status),
)
// Transcript (and therefore highlight) is only available once the
// pipeline reaches these two statuses; everything that depends on
// transcript.txt/json actually existing on disk is gated on this.
const hasTranscript = computed(() => ['menunggu_highlight', 'siap_premiere'].includes(project.value?.status))

async function load() {
  loading.value = true
  try {
    const [p, g] = await Promise.all([getProject(route.params.id), listGameModes()])
    project.value = p
    gameModes.value = g
    if (project.value.transcript_lang) {
      transcriptLang.value = project.value.transcript_lang
    }
    if (pastDownload.value) {
      promptText.value = (await getPrompt(route.params.id)).prompt
    }
    if (hasTranscript.value) {
      await loadHighlight()
    }
  } catch (err) {
    error(err.message)
  } finally {
    loading.value = false
  }
}

async function loadHighlight() {
  try {
    savedHighlight.value = await getHighlight(route.params.id)
  } catch (err) {
    if (err.code !== 'highlight_missing') {
      error(err.message)
    }
    savedHighlight.value = null
  }
}

async function submitHighlight() {
  savingHighlight.value = true
  highlightErrors.value = []
  try {
    savedHighlight.value = await saveHighlight(route.params.id, highlightBody.value)
    project.value = await getProject(route.params.id)
    success('Highlight tersimpan')
  } catch (err) {
    if (err.code === 'highlight_invalid' && err.details) {
      highlightErrors.value = err.details
    }
    error(err.message)
  } finally {
    savingHighlight.value = false
  }
}

async function removeHighlight() {
  const ok = await confirm({
    title: 'Hapus highlight?',
    text: 'highlight.json dan narasi.txt akan dihapus permanen.',
  })
  if (!ok) return
  deletingHighlight.value = true
  try {
    await deleteHighlight(route.params.id)
    savedHighlight.value = null
    project.value = await getProject(route.params.id)
    success('Highlight dihapus')
  } catch (err) {
    error(err.message)
  } finally {
    deletingHighlight.value = false
  }
}

async function openProjectFolder() {
  openingFolder.value = true
  try {
    await openFolder(route.params.id)
  } catch (err) {
    error(err.message)
  } finally {
    openingFolder.value = false
  }
}

function formatHMS(totalSec) {
  const total = Math.round(totalSec || 0)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const pad = (v) => String(v).padStart(2, '0')
  return `${pad(h)}:${pad(m)}:${pad(s)}`
}

function parseHMS(s) {
  const [h, m, sec] = s.split(':').map(Number)
  return h * 3600 + m * 60 + sec
}

function segmentDurationSec(s) {
  return Math.round(parseHMS(s.selesai) - parseHMS(s.mulai))
}

// Plays the video from segment s's "mulai" and stops it at "selesai" —
// preview only, no editing. onVideoPause clears the active marker so it
// only shows while actually playing (not once it stops at the boundary
// or the user pauses manually).
function playSegment(index, s) {
  const video = videoEl.value
  if (!video) return
  stopAtSeconds = parseHMS(s.selesai)
  activeSegmentIndex.value = index
  video.currentTime = parseHMS(s.mulai)
  video.play()
}

function onVideoTimeUpdate() {
  const video = videoEl.value
  if (stopAtSeconds !== null && video && video.currentTime >= stopAtSeconds) {
    video.pause()
    stopAtSeconds = null
  }
}

function onVideoPause() {
  activeSegmentIndex.value = -1
}

// savedHighlight.peringatan uses the same 1-based "segmen" numbering as
// the validation error details.
function warningsForSegment(index) {
  return (savedHighlight.value?.peringatan || []).filter((w) => w.segmen === index + 1)
}

async function copyPrompt() {
  try {
    await navigator.clipboard.writeText(promptText.value)
    success('Prompt disalin')
  } catch (err) {
    error('Gagal menyalin prompt')
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

async function retryTranscript() {
  retryingTranscript.value = true
  // Clear the old error and show the new state immediately: the
  // request's own response can still reflect the pre-retry status
  // (the worker hasn't picked the job up yet when it returns), so
  // don't wait for it or for the next SSE event to update the screen.
  if (project.value) {
    project.value = { ...project.value, status: 'transcript', error_message: '' }
  }
  try {
    await retryTranscribe(route.params.id, transcriptLang.value)
    success('Transkrip diulang')
  } catch (err) {
    error(err.message)
    await load() // the retry itself was rejected (e.g. job_running); restore the real state
  } finally {
    retryingTranscript.value = false
  }
}

async function cancelCurrentJob() {
  if (!job.value?.id) return
  cancellingJob.value = true
  try {
    await cancelJob(job.value.id)
  } catch (err) {
    error(err.message)
  } finally {
    cancellingJob.value = false
  }
}

function onJobEvent(ev) {
  if (ev.type === 'progress') {
    job.value = { id: ev.job_id, type: ev.job_type, progress: ev.progress, message: ev.message }
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
  <AppLayout>
    <RouterLink to="/" class="text-sm text-emerald-400 underline">&larr; Kembali ke daftar project</RouterLink>

    <p v-if="loading" class="text-slate-400">Memuat...</p>
    <template v-else-if="project">
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-semibold">{{ project.name }}</h1>
        <button
          :disabled="openingFolder"
          class="rounded bg-slate-800 px-3 py-1.5 text-sm font-medium disabled:opacity-50"
          @click="openProjectFolder"
        >
          Buka folder project
        </button>
      </div>
      <dl class="grid grid-cols-2 gap-2 text-sm">
        <dt class="text-slate-400">Game</dt>
        <dd>{{ gameName(gameModes, project.game_code) }}</dd>
        <dt class="text-slate-400">Status</dt>
        <dd :class="statusLabel(project.status).color">{{ statusLabel(project.status).text }}</dd>
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
          <span>{{ jobTypeLabel(job.type) }}</span>
          <span>{{ Math.round(job.progress) }}%</span>
        </div>
        <div class="h-2 w-full overflow-hidden rounded bg-slate-800">
          <div class="h-full bg-emerald-500 transition-all" :style="{ width: job.progress + '%' }"></div>
        </div>
        <p v-if="job.type === 'transcribe' && job.progress === 0" class="text-xs text-slate-500">Memuat model...</p>
        <p v-else-if="job.message" class="text-xs text-slate-500">{{ job.message }}</p>
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

      <section v-if="pastDownload" class="flex flex-col gap-3 border-t border-slate-800 pt-4">
        <h2 class="text-lg font-semibold">Transcript</h2>

        <p v-if="project.status === 'gagal_transcript'" class="text-sm text-rose-300">
          {{ project.error_message || 'Transcript gagal.' }}
        </p>

        <div class="flex items-end gap-2">
          <label class="flex flex-col gap-1">
            <span class="text-sm text-slate-400">Bahasa (auto, id, en, tl, ...)</span>
            <input v-model="transcriptLang" class="rounded bg-slate-800 px-3 py-2" />
          </label>
          <button
            v-if="!job"
            :disabled="retryingTranscript"
            class="rounded bg-emerald-600 px-3 py-2 text-sm font-medium disabled:opacity-50"
            @click="retryTranscript"
          >
            {{ retryingTranscript ? 'Memproses...' : 'Transkrip ulang' }}
          </button>
          <button
            v-else
            :disabled="cancellingJob"
            class="rounded bg-rose-700 px-3 py-2 text-sm font-medium disabled:opacity-50"
            @click="cancelCurrentJob"
          >
            {{ cancellingJob ? 'Membatalkan...' : 'Batalkan' }}
          </button>
        </div>

        <div v-if="hasTranscript" class="flex gap-3 text-sm">
          <a :href="transcriptUrl(project.id, 'txt')" class="text-emerald-400 underline">Unduh transcript.txt</a>
          <a :href="transcriptUrl(project.id, 'json')" class="text-emerald-400 underline">Unduh transcript.json</a>
        </div>
      </section>

      <section v-if="hasTranscript && promptText" class="flex flex-col gap-2 border-t border-slate-800 pt-4">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold">Prompt</h2>
          <button class="rounded bg-emerald-600 px-3 py-1.5 text-sm font-medium" @click="copyPrompt">Salin prompt</button>
        </div>
        <textarea
          :value="promptText"
          readonly
          rows="10"
          class="rounded bg-slate-900 p-3 text-xs text-slate-300"
        ></textarea>
      </section>

      <section v-if="hasTranscript" class="flex flex-col gap-3 border-t border-slate-800 pt-4">
        <h2 class="text-lg font-semibold">Highlight</h2>

        <template v-if="savedHighlight">
          <video
            ref="videoEl"
            :src="videoUrl(project.id)"
            controls
            class="w-full rounded bg-black"
            @timeupdate="onVideoTimeUpdate"
            @pause="onVideoPause"
          ></video>

          <p class="text-sm text-slate-400">
            {{ savedHighlight.segmen.length }} segmen · total durasi highlight {{ formatHMS(savedHighlight.total_durasi_sec) }}
          </p>
          <ul class="flex flex-col gap-2 text-sm">
            <li
              v-for="(s, i) in savedHighlight.segmen"
              :key="i"
              class="cursor-pointer rounded px-3 py-2 transition"
              :class="i === activeSegmentIndex ? 'bg-emerald-900/50 ring-1 ring-emerald-500' : 'bg-slate-900 hover:bg-slate-800'"
              @click="playSegment(i, s)"
            >
              <p class="font-medium">
                {{ i + 1 }}. [{{ s.mulai }} - {{ s.selesai }}] {{ s.label }}
                <span class="rounded px-1.5 py-0.5 text-xs" :class="categoryColor(s.kategori)">{{ s.kategori }}</span>
                <span class="text-xs text-slate-500">· {{ segmentDurationSec(s) }} detik</span>
                <span v-if="i === activeSegmentIndex" class="text-xs text-emerald-400">&#9654; sedang diputar</span>
              </p>
              <p class="text-xs text-slate-400">{{ s.narasi }}</p>
              <p v-for="(w, wi) in warningsForSegment(i)" :key="wi" class="text-xs text-amber-400">&#9888; {{ w.message }}</p>
            </li>
          </ul>
          <div class="flex gap-4 text-sm">
            <a :href="narasiUrl(project.id)" class="text-emerald-400 underline">Unduh narasi.txt</a>
            <button class="text-slate-400 underline" @click="savedHighlight = null">Tempel ulang</button>
            <button
              :disabled="deletingHighlight"
              class="text-rose-400 underline disabled:opacity-50"
              @click="removeHighlight"
            >
              {{ deletingHighlight ? 'Menghapus...' : 'Hapus highlight' }}
            </button>
          </div>
        </template>

        <template v-else>
          <p class="text-sm text-slate-400">Tempel jawaban JSON dari AI (boleh dalam blok kode ```json).</p>
          <textarea
            v-model="highlightBody"
            rows="10"
            class="rounded bg-slate-900 p-3 font-mono text-xs"
          ></textarea>
          <button
            :disabled="savingHighlight"
            class="self-start rounded bg-emerald-600 px-3 py-2 text-sm font-medium disabled:opacity-50"
            @click="submitHighlight"
          >
            {{ savingHighlight ? 'Memvalidasi...' : 'Validasi & Simpan' }}
          </button>
          <ul v-if="highlightErrors.length" class="flex flex-col gap-1 rounded border border-rose-800 bg-rose-950/40 p-3 text-sm text-rose-300">
            <li v-for="(e, i) in highlightErrors" :key="i">Segmen {{ e.segmen || '-' }} ({{ e.field }}): {{ e.message }}</li>
          </ul>
        </template>
      </section>
    </template>
  </AppLayout>
</template>
