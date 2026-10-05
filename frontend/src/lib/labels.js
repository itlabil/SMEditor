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
