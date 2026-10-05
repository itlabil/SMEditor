<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { getProject, getPrompt, openFolder, exportToFolder, retryDownload, retryTranscribe, transcriptUrl, videoUrl } from '../api/projects'
import { listGameModes, getPromptBlock } from '../api/promptBlocks'
import { saveHighlight, getHighlight, deleteHighlight, narasiUrl, updateSegment, deleteSegment, updateDraft } from '../api/highlight'
import { cancelJob } from '../api/jobs'
import { getSettings } from '../api/settings'
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
const folderPath = ref('')
const exportDestDir = ref('')
const exporting = ref(false)
const exportResultPath = ref('')
let pendingExportTarget = ''

const highlightBody = ref('')
const highlightErrors = ref([])
const savingHighlight = ref(false)
const deletingHighlight = ref(false)
const savedHighlight = ref(null)
const categories = ref([])

// Segment edit form (SM-15): only one segment is edited at a time.
const editingIndex = ref(-1)
const segmentForm = ref({ label: '', narasi: '', kategori: '', mulai: '', selesai: '' })
const segmentFormErrors = ref([])
const savingSegment = ref(false)
const deletingSegmentIndex = ref(-1)

// Draft result card (SM-16). Hero lists are edited as comma-separated text.
const editingDraft = ref(false)
const draftForm = ref(null)
const draftErrors = ref([])
const savingDraft = ref(false)
const DRAFT_TEAMS = [
  { key: 'tim_a', fallback: 'Tim A' },
  { key: 'tim_b', fallback: 'Tim B' },
]

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
const isMoba = computed(() => gameModes.value.find((g) => g.code === project.value?.game_code)?.genre === 'moba')
// The draft card shows for any highlight that has a draft, and for MOBA
// highlights without one so it can still be filled in by hand.
const showDraftCard = computed(() => !!savedHighlight.value?.draft || isMoba.value)

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
  await loadCategories()
}

// Valid "kategori" values for the project's genre, for the segment edit
// form. Non-critical: without them the field falls back to free text.
async function loadCategories() {
  const mode = gameModes.value.find((g) => g.code === project.value?.game_code)
  if (!mode) return
  try {
    categories.value = (await getPromptBlock(mode.block_code)).categories || []
  } catch (err) {
    categories.value = []
  }
}

function startEditSegment(index, s) {
  editingIndex.value = index
  segmentFormErrors.value = []
  segmentForm.value = { label: s.label, narasi: s.narasi, kategori: s.kategori, mulai: s.mulai, selesai: s.selesai }
}

function cancelEditSegment() {
  editingIndex.value = -1
  segmentFormErrors.value = []
}

async function submitSegment() {
  savingSegment.value = true
  segmentFormErrors.value = []
  try {
    savedHighlight.value = await updateSegment(route.params.id, editingIndex.value + 1, segmentForm.value)
    editingIndex.value = -1
    success('Segmen disimpan')
  } catch (err) {
    // Validation errors stay in the form; nothing was written on the server.
    if (err.code === 'highlight_invalid' && err.details) {
      segmentFormErrors.value = err.details
    } else {
      segmentFormErrors.value = [{ segmen: 0, field: '', message: err.message }]
    }
  } finally {
    savingSegment.value = false
  }
}

function teamDraft(key) {
  return savedHighlight.value?.draft?.[key] || { nama: '', pick: [], ban: [] }
}

function startEditDraft() {
  const form = {}
  for (const { key } of DRAFT_TEAMS) {
    const t = teamDraft(key)
    form[key] = { nama: t.nama || '', pick: (t.pick || []).join(', '), ban: (t.ban || []).join(', ') }
  }
  draftForm.value = form
  draftErrors.value = []
  editingDraft.value = true
}

function cancelEditDraft() {
  editingDraft.value = false
  draftErrors.value = []
}

function splitHeroes(text) {
  return text
    .split(',')
    .map((h) => h.trim())
    .filter(Boolean)
}

async function submitDraft() {
  savingDraft.value = true
  draftErrors.value = []
  const body = {}
  for (const { key } of DRAFT_TEAMS) {
    const t = draftForm.value[key]
    body[key] = { nama: t.nama.trim(), pick: splitHeroes(t.pick), ban: splitHeroes(t.ban) }
  }
  try {
    savedHighlight.value = await updateDraft(route.params.id, body)
    editingDraft.value = false
    success('Hasil draft disimpan')
  } catch (err) {
    // Validation errors stay in the form; nothing was written on the server.
    draftErrors.value = err.code === 'highlight_invalid' && err.details ? err.details : [{ field: '', message: err.message }]
  } finally {
    savingDraft.value = false
  }
}

async function removeSegment(index, s) {
  const ok = await confirm({
    title: `Hapus segmen ${index + 1}?`,
    text: `"${s.label}" akan dihapus dari highlight.json dan narasi.txt.`,
    confirmButtonText: 'Ya, hapus',
  })
  if (!ok) return
  deletingSegmentIndex.value = index
  try {
    savedHighlight.value = await deleteSegment(route.params.id, index + 1)
    if (editingIndex.value === index) cancelEditSegment()
    else if (editingIndex.value > index) editingIndex.value -= 1
    success('Segmen dihapus')
  } catch (err) {
    const detail = err.details?.[0]?.message
    error(detail ? `${err.message}: ${detail}` : err.message)
  } finally {
    deletingSegmentIndex.value = -1
  }
}

async function submitHighlight() {
  savingHighlight.value = true
  highlightErrors.value = []
  try {
    savedHighlight.value = await saveHighlight(route.params.id, highlightBody.value)
    cancelEditSegment()
    cancelEditDraft()
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
    const res = await openFolder(route.params.id)
    folderPath.value = res.path
    success('Folder project dibuka')
  } catch (err) {
    // Even when the file manager itself failed to launch, the backend
    // still reports the absolute path (err.details.path) so the user can
    // copy it and open it by hand.
    if (err.details?.path) {
      folderPath.value = err.details.path
    }
    error(err.message)
  } finally {
    openingFolder.value = false
  }
}

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text)
    success('Path disalin')
  } catch (err) {
    error('Gagal menyalin path')
  }
}

async function startExport(overwrite = false) {
  exporting.value = true
  try {
    const res = await exportToFolder(route.params.id, exportDestDir.value, overwrite)
    pendingExportTarget = res.target_dir
    exportResultPath.value = ''
    success('Menyalin ke folder dimulai')
  } catch (err) {
    if (err.code === 'export_dir_exists') {
      exporting.value = false
      const ok = await confirm({
        title: 'Folder tujuan sudah ada',
        text: `${err.details?.target_dir || 'Folder tujuan'} sudah ada. Timpa isinya?`,
      })
      if (ok) await startExport(true)
      return
    }
    error(err.message)
  } finally {
    exporting.value = false
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
  job.value = null
  // export never changes the project's status (see job_sync.go), so
  // there is nothing to reload; just report the outcome directly.
  if (ev.job_type === 'export') {
    if (ev.type === 'done') {
      exportResultPath.value = pendingExportTarget
      success('Selesai disalin ke folder')
    } else if (ev.type === 'failed') {
      error(ev.message || 'Salin ke folder gagal')
    }
    return
  }
  // done, failed, canceled: reload from the API instead of guessing the
  // new project status client-side.
  load()
}

const sse = useSSE(route.params.id, onJobEvent)

async function loadExportDefault() {
  try {
    const s = await getSettings()
    exportDestDir.value = s.export_dir || ''
  } catch (err) {
    // non-critical prefill; the field just stays empty
  }
}

onMounted(() => {
  load()
  loadExportDefault()
})
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
      <div v-if="folderPath" class="flex items-center gap-2 rounded bg-slate-900 px-3 py-2 text-xs">
        <code class="flex-1 overflow-x-auto whitespace-nowrap text-slate-300">{{ folderPath }}</code>
        <button class="shrink-0 rounded bg-slate-800 px-2 py-1 font-medium" @click="copyText(folderPath)">Salin</button>
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

          <div v-if="showDraftCard" class="flex flex-col gap-2 rounded bg-slate-900 px-3 py-2 text-sm">
            <div class="flex items-center justify-between">
              <h3 class="font-medium">Hasil draft</h3>
              <button
                v-if="!editingDraft"
                class="rounded bg-slate-800 px-2 py-1 text-xs hover:bg-slate-700"
                @click="startEditDraft"
              >
                Ubah
              </button>
            </div>

            <form v-if="editingDraft" class="flex flex-col gap-3" @submit.prevent="submitDraft">
              <div class="grid gap-3 sm:grid-cols-2">
                <fieldset v-for="team in DRAFT_TEAMS" :key="team.key" class="flex flex-col gap-2">
                  <label class="flex flex-col gap-1">
                    <span class="text-xs text-slate-400">Nama {{ team.fallback }}</span>
                    <input v-model="draftForm[team.key].nama" class="rounded bg-slate-800 px-3 py-1.5" />
                  </label>
                  <label class="flex flex-col gap-1">
                    <span class="text-xs text-slate-400">Pick, urut, pisahkan dengan koma (maks. 5)</span>
                    <input v-model="draftForm[team.key].pick" class="rounded bg-slate-800 px-3 py-1.5" />
                  </label>
                  <label class="flex flex-col gap-1">
                    <span class="text-xs text-slate-400">Ban, pisahkan dengan koma (boleh kosong)</span>
                    <input v-model="draftForm[team.key].ban" class="rounded bg-slate-800 px-3 py-1.5" />
                  </label>
                </fieldset>
              </div>
              <ul
                v-if="draftErrors.length"
                class="flex flex-col gap-1 rounded border border-rose-800 bg-rose-950/40 p-2 text-xs text-rose-300"
              >
                <li v-for="(e, ei) in draftErrors" :key="ei">
                  <template v-if="e.segmen">Segmen {{ e.segmen }}: </template>{{ e.field ? `${e.field}: ` : '' }}{{ e.message }}
                </li>
              </ul>
              <div class="flex gap-2">
                <button
                  type="submit"
                  :disabled="savingDraft"
                  class="rounded bg-emerald-600 px-3 py-1.5 text-sm font-medium disabled:opacity-50"
                >
                  {{ savingDraft ? 'Menyimpan...' : 'Simpan' }}
                </button>
                <button type="button" class="rounded bg-slate-800 px-3 py-1.5 text-sm" @click="cancelEditDraft">Batal</button>
              </div>
            </form>

            <p v-else-if="!savedHighlight.draft" class="text-xs text-slate-500">
              Belum ada hasil draft di highlight ini. Klik Ubah untuk mengisinya.
            </p>
            <div v-else class="grid gap-3 sm:grid-cols-2">
              <div v-for="team in DRAFT_TEAMS" :key="team.key" class="flex flex-col gap-1">
                <p class="font-medium">{{ teamDraft(team.key).nama || team.fallback }}</p>
                <p class="text-xs">
                  <span class="text-slate-400">Pick:</span>
                  {{ teamDraft(team.key).pick?.length ? teamDraft(team.key).pick.join(', ') : '-' }}
                </p>
                <p class="text-xs">
                  <span class="text-slate-400">Ban:</span>
                  {{ teamDraft(team.key).ban?.length ? teamDraft(team.key).ban.join(', ') : '-' }}
                </p>
              </div>
            </div>
          </div>

          <p class="text-sm text-slate-400">
            {{ savedHighlight.segmen.length }} segmen · total durasi highlight {{ formatHMS(savedHighlight.total_durasi_sec) }}
          </p>
          <ul class="flex flex-col gap-2 text-sm">
            <li
              v-for="(s, i) in savedHighlight.segmen"
              :key="i"
              class="rounded px-3 py-2 transition"
              :class="
                i === editingIndex
                  ? 'bg-slate-900 ring-1 ring-slate-600'
                  : i === activeSegmentIndex
                    ? 'cursor-pointer bg-emerald-900/50 ring-1 ring-emerald-500'
                    : 'cursor-pointer bg-slate-900 hover:bg-slate-800'
              "
              @click="i !== editingIndex && playSegment(i, s)"
            >
              <form v-if="i === editingIndex" class="flex flex-col gap-2" @submit.prevent="submitSegment">
                <p class="font-medium">Ubah segmen {{ i + 1 }}</p>
                <label class="flex flex-col gap-1">
                  <span class="text-xs text-slate-400">Label</span>
                  <input v-model="segmentForm.label" class="rounded bg-slate-800 px-3 py-1.5" />
                </label>
                <label class="flex flex-col gap-1">
                  <span class="text-xs text-slate-400">Narasi</span>
                  <textarea v-model="segmentForm.narasi" rows="3" class="rounded bg-slate-800 px-3 py-1.5"></textarea>
                </label>
                <div class="grid grid-cols-3 gap-2">
                  <label class="flex flex-col gap-1">
                    <span class="text-xs text-slate-400">Kategori</span>
                    <select v-if="categories.length" v-model="segmentForm.kategori" class="rounded bg-slate-800 px-3 py-1.5">
                      <option v-if="!categories.includes(segmentForm.kategori)" :value="segmentForm.kategori">{{ segmentForm.kategori }}</option>
                      <option v-for="c in categories" :key="c" :value="c">{{ c }}</option>
                    </select>
                    <input v-else v-model="segmentForm.kategori" class="rounded bg-slate-800 px-3 py-1.5" />
                  </label>
                  <label class="flex flex-col gap-1">
                    <span class="text-xs text-slate-400">Mulai (HH:MM:SS)</span>
                    <input v-model="segmentForm.mulai" class="rounded bg-slate-800 px-3 py-1.5 font-mono" />
                  </label>
                  <label class="flex flex-col gap-1">
                    <span class="text-xs text-slate-400">Selesai (HH:MM:SS)</span>
                    <input v-model="segmentForm.selesai" class="rounded bg-slate-800 px-3 py-1.5 font-mono" />
                  </label>
                </div>
                <ul
                  v-if="segmentFormErrors.length"
                  class="flex flex-col gap-1 rounded border border-rose-800 bg-rose-950/40 p-2 text-xs text-rose-300"
                >
                  <li v-for="(e, ei) in segmentFormErrors" :key="ei">
                    <template v-if="e.segmen">Segmen {{ e.segmen }}{{ e.field ? ` (${e.field})` : '' }}: </template>{{ e.message }}
                  </li>
                </ul>
                <div class="flex gap-2">
                  <button
                    type="submit"
                    :disabled="savingSegment"
                    class="rounded bg-emerald-600 px-3 py-1.5 text-sm font-medium disabled:opacity-50"
                  >
                    {{ savingSegment ? 'Menyimpan...' : 'Simpan' }}
                  </button>
                  <button type="button" class="rounded bg-slate-800 px-3 py-1.5 text-sm" @click="cancelEditSegment">Batal</button>
                </div>
              </form>
              <template v-else>
                <div class="flex items-start justify-between gap-2">
                  <p class="font-medium">
                    {{ i + 1 }}. [{{ s.mulai }} - {{ s.selesai }}] {{ s.label }}
                    <span class="rounded px-1.5 py-0.5 text-xs" :class="categoryColor(s.kategori)">{{ s.kategori }}</span>
                    <span class="text-xs text-slate-500">· {{ segmentDurationSec(s) }} detik</span>
                    <span v-if="i === activeSegmentIndex" class="text-xs text-emerald-400">&#9654; sedang diputar</span>
                  </p>
                  <div class="flex shrink-0 gap-2 text-xs">
                    <button class="rounded bg-slate-800 px-2 py-1 hover:bg-slate-700" @click.stop="startEditSegment(i, s)">Ubah</button>
                    <button
                      :disabled="deletingSegmentIndex === i"
                      class="rounded bg-slate-800 px-2 py-1 text-rose-400 hover:bg-slate-700 disabled:opacity-50"
                      @click.stop="removeSegment(i, s)"
                    >
                      {{ deletingSegmentIndex === i ? 'Menghapus...' : 'Hapus' }}
                    </button>
                  </div>
                </div>
                <p class="text-xs text-slate-400">{{ s.narasi }}</p>
                <p v-for="(w, wi) in warningsForSegment(i)" :key="wi" class="text-xs text-amber-400">&#9888; {{ w.message }}</p>
              </template>
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

      <section v-if="hasTranscript" class="flex flex-col gap-3 border-t border-slate-800 pt-4">
        <h2 class="text-lg font-semibold">Salin ke folder</h2>
        <p class="text-sm text-slate-400">
          Menyalin video, highlight.json, dan narasi.txt ke folder kerja Premiere, supaya project ini boleh dihapus
          tanpa membuat media offline.
        </p>
        <label class="flex flex-col gap-1">
          <span class="text-sm text-slate-400">Folder tujuan</span>
          <input v-model="exportDestDir" class="rounded bg-slate-800 px-3 py-2" placeholder="/path/ke/folder/premiere" />
        </label>
        <div>
          <button
            v-if="!(job && job.type === 'export')"
            :disabled="project.status !== 'siap_premiere' || exporting || !!job"
            class="rounded bg-emerald-600 px-3 py-2 text-sm font-medium disabled:opacity-50"
            @click="startExport()"
          >
            {{ exporting ? 'Memulai...' : 'Salin ke folder' }}
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
        <div v-if="exportResultPath" class="flex items-center gap-2 rounded bg-slate-900 px-3 py-2 text-xs">
          <code class="flex-1 overflow-x-auto whitespace-nowrap text-slate-300">{{ exportResultPath }}</code>
          <button class="shrink-0 rounded bg-slate-800 px-2 py-1 font-medium" @click="copyText(exportResultPath)">Salin</button>
        </div>
      </section>
    </template>
  </AppLayout>
</template>
