# Konvensi

## Umum

- Kerjakan satu tiket dari `docs/tickets.md` per sesi. Jangan mengerjakan tiket berikutnya tanpa diminta.
- Jangan menambah dependensi tanpa menyebutkan alasannya lebih dulu.
- Jika kebutuhan tidak jelas, tanyakan; jangan menebak.
- Teks antarmuka berbahasa Indonesia. Nama variabel, fungsi, dan komentar kode berbahasa Inggris.

## Go

- Respons API: `{"data": ...}` atau `{"error": {"code": "...", "message": "..."}}`, selalu JSON, selalu lewat helper `httpx`.
- Kode error berupa snake_case tetap (`invalid_url`, `project_not_found`); pesan berbahasa Indonesia.
- Error dibungkus dengan `fmt.Errorf("...: %w", err)`. Tidak ada `panic` di luar `main`.
- Semua fungsi yang menyentuh DB atau proses luar menerima `context.Context` sebagai parameter pertama.
- ID memakai ULID. Waktu disimpan UTC.
- SQLite memakai `modernc.org/sqlite`. Tidak boleh ada dependensi CGO.
- Proses luar dijalankan dengan `exec.CommandContext` dan argumen terpisah; tidak pernah lewat shell.
- Path selalu disusun dengan `filepath.Join`. Tidak ada path atau nama executable yang ditulis tetap; ambil dari settings.
- Tes memakai table-driven test dari paket `testing`.

## Vue

- `<script setup>` dan Composition API.
- Semua panggilan HTTP lewat `src/api/`; komponen tidak memanggil `fetch` langsung.
- Styling hanya dengan kelas Tailwind.
- Konfirmasi dan notifikasi lewat SweetAlert2 melalui composable, bukan `alert()` atau `confirm()`.
- Setiap aksi merusak (hapus project, ganti highlight) wajib konfirmasi.

## Plugin Premiere

- Panel (JS modern) dan skrip host (ExtendScript, setara ES3) dipisah tegas.
- Di ExtendScript: hanya `var`, tanpa arrow function, tanpa `let`/`const`, tanpa template string.
- Data dari panel ke host dikirim sebagai string JSON; host mengembalikan string JSON.
- Setiap fungsi host dibungkus `try/catch` dan mengembalikan `{ok, data, error}`.

## Keamanan

- Tolak URL selain domain YouTube sebelum memanggil yt-dlp.
- ID project dari URL divalidasi sebagai ULID sebelum dipakai menyusun path.
- Hapus folder hanya boleh menyasar path di dalam `data/projects/`.
