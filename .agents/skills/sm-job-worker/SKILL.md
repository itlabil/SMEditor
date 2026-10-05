---
name: sm-job-worker
description: Cara kerja antrean job, worker, pembatalan, dan progres SSE di SMEditor. Pakai saat menambah jenis job baru, mengubah worker, atau menulis kode yang melaporkan progres ke browser.
---

# Job dan worker

Pekerjaan panjang (unduh, konversi, transcript) tidak pernah dijalankan di dalam handler HTTP. Handler hanya mengantrekan job; satu worker di latar belakang yang menjalankannya.

## Aturan dasar

- Hanya ada **satu** goroutine worker, dan ia menjalankan **satu** job pada satu waktu. Whisper dan ffmpeg sama-sama berat; menjalankannya bersamaan di PC 8 GB akan gagal.
- Urutan: job `queued` dengan `created_at` tertua lebih dulu.
- Sumber kebenaran status adalah tabel `jobs`. Worker tidak menyimpan antrean di memori; ia hanya menerima sinyal "ada job baru" lewat channel lalu membaca DB.
- Worker juga memeriksa DB setiap beberapa detik, supaya job tidak tertinggal jika sinyal terlewat.

## Siklus hidup job

```
queued -> running -> done
                  -> failed
                  -> canceled
```

Setiap perubahan status job disertai perubahan status project yang sesuai (lihat tabel status di `docs/flow.md`), dalam satu transaksi.

| Job | Saat mulai | Saat done | Saat failed/canceled |
|---|---|---|---|
| `download` | project `mengunduh` | antrekan `convert` (jika perlu) lalu `transcribe` | project `gagal_unduh` |
| `convert` | tetap `mengunduh` | lanjut ke `transcribe` | project `gagal_unduh` |
| `transcribe` | project `transcript` | project `menunggu_highlight` | project `gagal_transcript` |
| `export` | tidak diubah | tidak diubah | tidak diubah |

`export` ("Salin ke folder", lihat `docs/prd.md`) sengaja tidak masuk ke switch start/fail/done manapun di `project.JobSync`: ia tidak pernah mengubah status project, sukses maupun gagal. Ia juga satu-satunya jenis job yang memakai kolom `jobs.payload` (string bebas; untuk `export` isinya folder tujuan absolut, dihitung sekali oleh handler sebelum job diantrekan). Job lain selalu mengantrekan dengan payload kosong.

Saat app start, sebelum worker jalan: semua job `running` diubah menjadi `failed` dengan error "app ditutup saat job berjalan", dan status project disesuaikan (kecuali `export`, yang tidak punya status project untuk disesuaikan).

## Menambah jenis job

Tiap jenis job adalah satu implementasi interface ini, didaftarkan ke worker di `internal/app`:

```go
type Runner interface {
    Type() string
    Run(ctx context.Context, job Job, report ProgressFunc) error
}

type ProgressFunc func(percent float64, message string)
```

- `Run` harus berhenti segera saat `ctx` dibatalkan dan mengembalikan `ctx.Err()`.
- `Run` tidak menyentuh tabel `jobs`; worker yang mencatat status berdasarkan nilai kembaliannya.
- Runner memanggil paket `internal/tools/*` untuk proses luar (lihat skill `sm-external-tools`).
- Runner harus aman diulang: file hasil ditulis ke nama sementara lalu di-rename, dan file sisa percobaan sebelumnya dihapus di awal.

## Pembatalan

- Worker menyimpan `cancel` dari context job yang sedang berjalan.
- `POST /api/jobs/:id/cancel`: jika job `queued`, langsung tandai `canceled`. Jika `running`, panggil `cancel` dan tunggu `Run` kembali.
- Menghapus project: batalkan semua job project itu dan **tunggu sampai proses anak benar-benar berhenti** sebelum menghapus folder. Di Windows, file yang masih dibuka proses lain tidak bisa dihapus.
- Error akibat pembatalan dicatat sebagai `canceled`, bukan `failed`.

## Progres

- `report` boleh dipanggil sesering mungkin; worker yang membatasi: tulis ke DB paling sering 1 kali per detik, kirim ke SSE paling sering 4 kali per detik.
- `percent` 0 sampai 100. Jika tidak diketahui, kirim nilai terakhir dan perbarui `message` saja.

## SSE

`GET /api/projects/:id/events` berlangganan ke hub di memori.

- Event: `{"type":"progress|done|failed|canceled","job_id":"...","job_type":"...","progress":42.5,"message":"..."}`.
- Saat klien tersambung, kirim dulu keadaan job terakhir project itu, supaya halaman yang baru dibuka langsung benar.
- Kirim komentar `: ping` setiap 15 detik agar koneksi tidak diputus.
- Hub tidak boleh memblokir worker: jika buffer klien penuh, event progres dibuang; event akhir (`done`, `failed`, `canceled`) tidak boleh dibuang.
- Handler berhenti saat `c.Request.Context()` selesai dan melepas langganannya.

Di frontend, pakai composable `useSSE(projectId)` berbasis `EventSource`. Saat menerima event akhir, muat ulang data project dari API; jangan menebak status baru di klien.

## Tes

- Worker dites dengan Runner palsu: urutan antrean, satu job pada satu waktu, pembatalan saat running, pemulihan job `running` saat start.
- Tidak ada tes yang memanggil yt-dlp, ffmpeg, atau Whisper sungguhan.
