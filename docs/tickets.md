# Tickets — SMEditor

Kerjakan berurutan, satu tiket per sesi. Sebuah tiket selesai jika semua kriteria terima terpenuhi dan Definition of Done di bawah terpenuhi.

## Definition of Done (berlaku untuk semua tiket)

- `go build ./...` dan `go vet ./...` lolos; `npm run build` lolos.
- Kode mengikuti `.agents/rules/`.
- Logika yang punya aturan (parser, validasi, perakit prompt) punya unit test.
- Diuji manual sesuai kriteria terima, lalu di-commit dengan pesan `SM-xx: ringkasan`.
- Jika perilaku berubah dari dokumen, dokumen di `docs/` ikut diperbarui.

## Ringkasan

| ID | Judul | Bergantung pada |
|---|---|---|
| SM-01 | Kerangka repo dan server | — |
| SM-02 | Database dan migrasi | SM-01 |
| SM-03 | Pengaturan dan pemeriksaan tool | SM-02 |
| SM-04 | CRUD project | SM-02 |
| SM-05 | Antrean job dan SSE | SM-04 |
| SM-06 | Unduh video | SM-03, SM-05 |
| SM-07 | Transcript | SM-06 |
| SM-08 | Mode game dan prompt | SM-04 |
| SM-09 | Validasi highlight dan narasi | SM-07, SM-08 |
| SM-10 | Build Windows | SM-09 |
| SM-11 | Plugin: panel dan baca file | — (Windows) |
| SM-12 | Plugin: sequence dan potongan | SM-11 |
| SM-13 | Plugin: marker dan laporan | SM-12 |
| SM-14 | Salin ke folder | SM-09 |

---

## SM-01 — Kerangka repo dan server

Membuat struktur `backend/`, `frontend/`, `premiere-plugin/`, `docs/`, `tools/`, `data/`.

Kriteria terima:
- `backend` memakai Go + Gin dengan struktur modul sesuai `architecture.md`.
- `frontend` memakai Vue 3 + Vite + Tailwind + SweetAlert2 + Vue Router.
- `GET /api/health` mengembalikan `{"data":{"status":"ok"}}`.
- Mode dev: Vite mem-proxy `/api` ke Go. Mode produksi: hasil build Vue ditanam dengan `go:embed` dan dilayani Go.
- Server hanya mendengarkan di `127.0.0.1`.
- `Makefile` punya target `dev`, `build`, `test`.
- `tools/` dan `data/` masuk `.gitignore`.

## SM-02 — Database dan migrasi

Kriteria terima:
- SQLite dengan `modernc.org/sqlite`, file di `data/app.db`, foreign key aktif.
- Migrasi berjalan otomatis saat start dan aman dijalankan berulang.
- Semua tabel di `erd.md` terbentuk.
- Seed: 12 mode game, 1 mode umum, kerangka prompt, dan 5 blok tugas, isinya dari `prd.md`.

## SM-03 — Pengaturan dan pemeriksaan tool

Kriteria terima:
- `GET/PUT /api/settings` membaca dan menyimpan semua key di `erd.md`.
- `GET /api/settings/check` melaporkan per tool: ditemukan atau tidak, dan versinya.
- Halaman Pengaturan menampilkan form dan hasil pemeriksaan.
- Nama file tool menyesuaikan OS (`.exe` di Windows) tanpa mengubah kode.

## SM-04 — CRUD project

Kriteria terima:
- Buat project: nama dan URL wajib; URL harus YouTube; selain itu ditolak dengan pesan jelas.
- Folder `data/projects/{id}/` dibuat saat project dibuat.
- Daftar project menampilkan nama, game, status, durasi, ukuran, thumbnail.
- Hapus project memakai konfirmasi SweetAlert, menghapus baris dan folder.
- Setelah dihapus, folder project benar-benar tidak ada di disk.

## SM-05 — Antrean job dan SSE

Kriteria terima:
- Satu worker memproses job `queued` tertua, satu per satu.
- Job bisa dibatalkan; proses anak ikut berhenti.
- Saat start, job `running` yang tertinggal ditandai `failed`.
- `GET /api/projects/:id/events` mengirim event `progress`, `done`, `failed`.
- Halaman project menampilkan progress bar yang bergerak tanpa refresh.
- Menghapus project yang jobnya berjalan menghentikan job lebih dulu.

## SM-06 — Unduh video

Kriteria terima:
- yt-dlp memilih H.264 + AAC MP4, 1080p lalu 720p.
- Progres persen, kecepatan, dan sisa waktu tampil.
- Setelah selesai, durasi, resolusi, fps, codec, ukuran, dan thumbnail tersimpan.
- Jika codec bukan H.264, video dikonversi ke H.264 dengan resolusi dan fps yang sama.
- URL yang tidak bisa diunduh menghasilkan status `gagal_unduh` dengan pesan dari yt-dlp, dan tombol Ulangi berfungsi.
- Parser baris progres yt-dlp punya unit test.

## SM-07 — Transcript

Kriteria terima:
- Audio diekstrak ke WAV 16 kHz mono.
- Bahasa bisa dipilih: otomatis, Indonesia, Inggris, Filipina, atau kode lain.
- `transcript.json` dan `transcript.txt` sesuai format di `prd.md`.
- Transcript berjalan dengan CPU di Ubuntu; pilihan GPU tersedia lewat pengaturan.
- Transcript bisa diulang dengan bahasa lain dan menimpa file lama.
- Konversi keluaran Whisper ke kedua format punya unit test.

## SM-08 — Mode game dan prompt

Kriteria terima:
- `GET /api/projects/:id/prompt` merakit kerangka + blok tugas + istilah game.
- Semua placeholder terisi; tidak ada `{...}` tersisa di hasil. Field opsional yang kosong diganti "tidak diisi".
- Halaman project menampilkan prompt dengan tombol Salin dan tombol unduh `transcript.txt`.
- Blok prompt bisa diedit di Pengaturan dan dikembalikan ke bawaan.
- Perakit prompt punya unit test untuk minimal satu game per genre.

## SM-09 — Validasi highlight dan narasi

Kriteria terima:
- Menerima JSON polos maupun yang terbungkus blok kode.
- Semua aturan di `flow.md` bagian 4.5 diterapkan; kesalahan dilaporkan per segmen dengan nomor segmen.
- Jika ada satu saja kesalahan, tidak ada file yang ditulis.
- Jika lolos: `highlight.json` (dengan `video` dan `durasi`) dan `narasi.txt` tertulis, status menjadi `siap_premiere`.
- Halaman menampilkan daftar segmen, total durasi highlight, dan tombol buka folder.
- Validator punya unit test untuk tiap aturan.

## SM-10 — Build Windows

Kriteria terima:
- `make build-windows` di Ubuntu menghasilkan `smeditor.exe` dengan frontend tertanam.
- `docs/setup-windows.md` menjelaskan tool yang harus diunduh dan letaknya di `tools/`.
- Di Windows: unduh, transcript dengan GPU, dan validasi berjalan dari awal sampai akhir.

## SM-11 — Plugin: panel dan baca file

Dikerjakan dan diuji di Windows.

Kriteria terima:
- Panel CEP muncul di menu Window > Extensions.
- Panel bisa memilih video dan `highlight.json`, lalu menampilkan jumlah segmen dan total durasi.
- JSON yang rusak atau segmen yang tidak valid dilaporkan di panel.
- `premiere-plugin/README.md` menjelaskan cara pasang dan mengaktifkan mode debug.

## SM-12 — Plugin: sequence dan potongan

Kriteria terima:
- Video diimpor jika belum ada di project Premiere.
- Sequence baru memakai resolusi dan fps video.
- Jumlah potongan di timeline sama dengan jumlah segmen yang valid, berurutan.
- Tiap potongan diperpanjang sesuai tambahan waktu (default 1 detik) dan dibatasi di awal dan akhir video.
- Antar potongan ada jeda sesuai pengaturan (default 1 detik).
- Video dan audio tetap tersambung di tiap potongan.

## SM-13 — Plugin: marker dan laporan

Kriteria terima:
- Tiap potongan punya marker: nama berisi label, komentar berisi narasi.
- Warna marker berbeda per kategori.
- Panel menampilkan laporan: jumlah potongan dibuat dan segmen yang dilewati beserta alasannya.
- Menjalankan ulang membuat sequence baru tanpa mengubah yang lama.

## SM-14 — Salin ke folder

Memindahkan bahan project ke folder kerja Premiere lewat app, supaya project di app boleh dihapus tanpa membuat media offline.

Kriteria terima:
- Tombol "Salin ke folder" di halaman project hanya aktif saat status `siap_premiere`.
- Pengguna mengisi path folder tujuan lewat input teks. Nilai terakhir disimpan sebagai setting baru `export_dir` dan menjadi isian default berikutnya.
- App membuat subfolder bernama project (dibersihkan dari karakter yang tidak sah di Windows dan Linux) di dalam tujuan, lalu menyalin video, `highlight.json`, dan `narasi.txt` ke sana. Field `video` di `highlight.json` hasil salinan harus sama dengan nama file video yang disalin.
- Penyalinan berjalan sebagai job jenis baru `export` lewat worker, dengan progres dan bisa dibatalkan, mengikuti skill `sm-job-worker`. Job ini tidak mengubah status project.
- Jika subfolder tujuan sudah ada, minta konfirmasi SweetAlert sebelum menimpa. Jika folder tujuan tidak ada atau tidak bisa ditulis, kembalikan error yang jelas. Salin ke file sementara lalu rename, dan bersihkan file setengah jadi saat batal atau gagal.
- Setelah selesai, tampilkan path hasil dengan tombol salin.
- `docs/prd.md`, `docs/flow.md`, `docs/erd.md`, `docs/sequence.md` diperbarui; kode error baru ditambahkan ke skill `sm-api-response`.
- Unit test untuk pembersihan nama folder dan penolakan path tujuan yang tidak sah.
