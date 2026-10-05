// Shared project/game_mode -> display label mapping, used by both the
// project list and the project detail page so they never drift apart.

const STATUS_LABELS = {
  baru: { text: 'Baru', color: 'text-slate-300' },
  mengunduh: { text: 'Mengunduh', color: 'text-amber-400' },
  gagal_unduh: { text: 'Unduh gagal', color: 'text-rose-400' },
  transcript: { text: 'Membuat transcript', color: 'text-amber-400' },
  gagal_transcript: { text: 'Transcript gagal', color: 'text-rose-400' },
  menunggu_highlight: { text: 'Menunggu highlight', color: 'text-slate-300' },
  siap_premiere: { text: 'Siap untuk Premiere', color: 'text-emerald-400' },
}

// statusLabel returns { text, color } for a project status code. Unknown
// codes fall back to showing the raw code so nothing ever disappears
// silently if a new status is added to the backend later.
export function statusLabel(status) {
  return STATUS_LABELS[status] || { text: status, color: 'text-slate-300' }
}

// gameName looks up a game_code's display name from the list returned by
// GET /api/game-modes. Falls back to the raw code if the list hasn't
// loaded yet or the code is unknown.
export function gameName(gameModes, code) {
  const found = gameModes.find((g) => g.code === code)
  return found ? found.name : code
}

const JOB_TYPE_LABELS = {
  download: 'Mengunduh video',
  convert: 'Mengonversi video',
  transcribe: 'Membuat transcript',
}

// jobTypeLabel returns a friendly label for a job's type (the SSE
// event's job_type), falling back to the raw type for anything new.
export function jobTypeLabel(type) {
  return JOB_TYPE_LABELS[type] || type
}

// Highlight categories aren't a small fixed set: each genre has its own
// list (docs/prd.md), and a block's categories can even be edited in
// Pengaturan. So instead of a lookup table, categoryColor hashes the
// category string to a consistent pick from a fixed palette - the same
// category always gets the same color within one highlight.
const CATEGORY_COLORS = [
  'bg-sky-900/60 text-sky-300',
  'bg-emerald-900/60 text-emerald-300',
  'bg-amber-900/60 text-amber-300',
  'bg-rose-900/60 text-rose-300',
  'bg-violet-900/60 text-violet-300',
  'bg-cyan-900/60 text-cyan-300',
  'bg-lime-900/60 text-lime-300',
  'bg-pink-900/60 text-pink-300',
]

export function categoryColor(kategori) {
  let hash = 0
  for (let i = 0; i < kategori.length; i++) {
    hash = (hash * 31 + kategori.charCodeAt(i)) % CATEGORY_COLORS.length
  }
  return CATEGORY_COLORS[Math.abs(hash) % CATEGORY_COLORS.length]
}
