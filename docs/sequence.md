# Sequence — SMEditor

Semua respons API berformat JSON: `{"data": ...}` saat berhasil, `{"error": {"code": "...", "message": "..."}}` saat gagal. Progres job dikirim lewat Server-Sent Events.

## Daftar endpoint

| Method | Path | Fungsi |
|---|---|---|
| GET | `/api/projects` | Daftar project |
| POST | `/api/projects` | Buat project, antrekan unduh |
| GET | `/api/projects/:id` | Detail project dan job terakhir |
| DELETE | `/api/projects/:id` | Hapus project dan semua filenya |
| GET | `/api/projects/:id/thumbnail` | Gambar thumbnail project |
| POST | `/api/projects/:id/download` | Ulangi unduh |
| POST | `/api/projects/:id/transcribe` | Ulangi transcript (body: bahasa) |
| POST | `/api/jobs/:id/cancel` | Batalkan job |
| GET | `/api/projects/:id/events` | SSE progres job project |
| GET | `/api/projects/:id/transcript?format=txt\|json` | Unduh transcript |
| GET | `/api/projects/:id/prompt` | Prompt yang sudah dirakit |
| POST | `/api/projects/:id/highlight` | Validasi dan simpan highlight |
| GET | `/api/projects/:id/highlight` | Baca highlight yang sudah tersimpan |
| DELETE | `/api/projects/:id/highlight` | Hapus highlight |
| GET | `/api/projects/:id/narasi` | Unduh `narasi.txt` |
| POST | `/api/projects/:id/open-folder` | Buka folder project di file manager |
| GET | `/api/game-modes` | Daftar mode game |
| GET, PUT | `/api/prompt-blocks/:code` | Baca dan ubah blok prompt |
| POST | `/api/prompt-blocks/:code/reset` | Kembalikan blok prompt ke bawaan |
| GET, PUT | `/api/settings` | Baca dan ubah pengaturan |
| GET | `/api/settings/check` | Periksa apakah tiap tool ditemukan |

## 1. Buat project dan unduh video

```mermaid
sequenceDiagram
    actor U as Pengguna
    participant FE as Vue
    participant API as Go API
    participant DB as SQLite
    participant W as Worker
    participant YT as yt-dlp
    participant FF as ffprobe/ffmpeg

    U->>FE: Isi form, klik Buat
    FE->>API: POST /api/projects
    API->>API: Validasi URL YouTube
    API->>DB: INSERT projects (status baru)
    API->>DB: INSERT jobs (download, queued)
    API-->>FE: 201 project
    FE->>API: GET /api/projects/:id/events (SSE)

    W->>DB: Ambil job queued tertua
    W->>DB: job running, project mengunduh
    W->>YT: Jalankan dengan format H.264 1080p/720p
    loop Selama mengunduh
        YT-->>W: Baris progres
        W->>DB: UPDATE jobs.progress
        W-->>FE: SSE progress
    end
    YT-->>W: Selesai
    W->>FF: ffprobe source
    FF-->>W: durasi, resolusi, fps, codec
    W->>FF: Ambil thumbnail
    W->>DB: UPDATE projects metadata
    alt Codec bukan H.264
        W->>DB: INSERT jobs (convert, queued)
    end
    W->>DB: job done, INSERT jobs (transcribe, queued)
    W-->>FE: SSE done
```

## 2. Transcript

```mermaid
sequenceDiagram
    participant FE as Vue
    participant W as Worker
    participant DB as SQLite
    participant FF as ffmpeg
    participant WH as Whisper
    participant FS as Folder project

    W->>DB: job running, project transcript
    W->>FF: Ekstrak audio WAV 16 kHz mono
    FF-->>FS: audio.wav
    W->>WH: Jalankan (model, bahasa, keluaran JSON)
    loop Selama transcript
        WH-->>W: Baris progres
        W-->>FE: SSE progress
    end
    WH-->>W: Segmen dengan waktu
    W->>FS: Tulis transcript.json dan transcript.txt
    W->>DB: job done, project menunggu_highlight
    W-->>FE: SSE done
```

## 3. Prompt dan validasi highlight

```mermaid
sequenceDiagram
    actor U as Pengguna
    participant FE as Vue
    participant API as Go API
    participant DB as SQLite
    participant FS as Folder project
    participant AI as AI di luar app

    FE->>API: GET /api/projects/:id/prompt
    API->>DB: Baca project, game_mode, prompt_blocks
    API->>API: Rakit frame + blok tugas + istilah
    API-->>FE: Teks prompt
    U->>FE: Salin prompt, unduh transcript.txt
    U->>AI: Tempel prompt dan transcript
    AI-->>U: JSON highlight
    U->>FE: Tempel JSON, klik Validasi
    FE->>API: POST /api/projects/:id/highlight
    API->>API: Buang pembungkus blok kode, parse, validasi
    alt Ada kesalahan
        API-->>FE: 422 daftar kesalahan per segmen
        FE-->>U: Tampilkan kesalahan
    else Lolos
        API->>FS: Tulis highlight.json dan narasi.txt
        API->>DB: project siap_premiere, has_highlight 1
        API-->>FE: 200 daftar segmen
    end
```

## 4. Hapus project

```mermaid
sequenceDiagram
    actor U as Pengguna
    participant FE as Vue
    participant API as Go API
    participant W as Worker
    participant DB as SQLite
    participant FS as Folder project

    U->>FE: Klik Hapus
    FE-->>U: Konfirmasi SweetAlert
    U->>FE: Ya, hapus
    FE->>API: DELETE /api/projects/:id
    API->>W: Batalkan job project ini
    W-->>API: Proses anak sudah berhenti
    API->>FS: Hapus folder project
    API->>DB: DELETE projects (jobs ikut terhapus)
    API-->>FE: 204
```

## 5. Plugin Premiere

```mermaid
sequenceDiagram
    actor U as Pengguna
    participant P as Panel (HTML/JS)
    participant H as Skrip host (ExtendScript)
    participant PR as Premiere

    U->>P: Pilih video dan highlight.json
    P->>P: Baca JSON, validasi ulang
    U->>P: Klik Buat Sequence
    P->>H: buatSequence(videoPath, segmen, pad, jeda)
    H->>PR: Cari video di project
    alt Belum ada
        H->>PR: Impor video
    end
    H->>PR: Buat sequence dari klip
    H->>PR: Kosongkan isi sequence
    loop Tiap segmen
        H->>H: in = max(0, mulai - pad), out = min(durasi, selesai + pad)
        H->>PR: Set in/out klip sumber
        H->>PR: Sisipkan ke V1/A1 di posisi sekarang
        H->>PR: Buat marker (label, narasi)
        H->>H: posisi += (out - in) + jeda
    end
    H-->>P: Jumlah potongan, segmen yang dilewati
    P-->>U: Tampilkan laporan
```
