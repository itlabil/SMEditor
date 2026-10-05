---
name: sm-external-tools
description: Cara memanggil yt-dlp, ffmpeg, ffprobe, dan Whisper dari SMEditor, membaca progresnya, dan menangani pembatalan di Linux dan Windows. Pakai saat menulis atau mengubah kode di backend/internal/tools/.
---

# Proses luar

yt-dlp, ffmpeg, ffprobe, dan Whisper adalah program terpisah. Semua pemanggilannya ada di `backend/internal/tools/<nama>/`, dan modul lain memakainya lewat interface.

## Aturan umum

- Lokasi program selalu lewat `tools.ResolveExecutable` dengan nilai dari settings. Tidak ada nama program yang ditulis tetap.
- Selalu `exec.CommandContext(ctx, path, args...)` dengan argumen terpisah. Tidak pernah lewat shell, tidak pernah menyambung string perintah.
- Nilai dari pengguna (URL, nama file) hanya boleh menjadi satu argumen utuh. Sebelum URL, sisipkan `--` agar tidak terbaca sebagai opsi.
- Sebelum menjalankan, jika program tidak ditemukan kembalikan `httpx.ErrConflict("tool_not_found", ...)` dengan nama tool di pesannya.
- Versi yang dipasang berbeda-beda. Sebelum memakai sebuah opsi, periksa `--help` program di mesin ini; jangan mengandalkan ingatan. Jika opsi tidak ada, laporkan, jangan menebak pengganti.

## Membaca keluaran

- Baca stdout dan stderr baris demi baris dengan `bufio.Scanner` di goroutine terpisah, dan habiskan keduanya sebelum `cmd.Wait()`.
- Banyak program menulis progres dengan `\r`, bukan `\n`. Pakai fungsi split yang memecah di keduanya.
- Simpan sekitar 20 baris terakhir stderr. Jika program gagal, baris itu menjadi isi `jobs.error`.
- Parser progres adalah fungsi murni `parseProgress(line string) (Progress, bool)` dengan unit test dari contoh baris asli. Baris yang tidak dikenali diabaikan, bukan error.

## Pembatalan

Saat `ctx` dibatalkan, seluruh pohon proses harus mati, karena yt-dlp memanggil ffmpeg sebagai anak.

- Linux: jalankan dengan `SysProcAttr{Setpgid: true}`, dan set `cmd.Cancel` untuk mengirim sinyal ke grup proses (`syscall.Kill(-pid, SIGTERM)`).
- Windows: buat proses dengan `CREATE_NEW_PROCESS_GROUP`, dan hentikan pohonnya dengan `taskkill /T /F /PID <pid>`.
- Set `cmd.WaitDelay` (misalnya 5 detik) supaya proses yang tidak mau berhenti dimatikan paksa.
- Kode khusus OS dipisah dengan build tag: `proc_unix.go` dan `proc_windows.go`. Keduanya harus tetap bisa di-compile (`GOOS=windows go build ./...`).
- Setelah batal atau gagal, hapus file setengah jadi (`*.part`, file sementara).

## yt-dlp

- Format: utamakan video H.264 (`avc1`) dan audio AAC (`mp4a`) dalam MP4, tinggi maksimal 1080; jika tidak ada, format terbaik sampai 1080.
- Pakai `--no-playlist`, `--newline` (progres per baris), dan `--progress-template` supaya baris progres punya bentuk tetap yang mudah di-parse.
- Arahkan lokasi ffmpeg dengan `--ffmpeg-location` dari settings.
- Keluaran ditulis ke nama sementara di folder project, lalu di-rename menjadi `source.mp4` setelah selesai.
- Ambil judul video dengan pemanggilan terpisah (`--print title` atau `--dump-json` dengan `--skip-download`) sebelum mengunduh.

## ffprobe

- Minta keluaran JSON (`-of json -show_format -show_streams`) dan parse ke struct. Jangan mem-parse teks biasa.
- fps berbentuk pecahan seperti `30000/1001`; ubah ke angka desimal.
- Yang disimpan: durasi, lebar, tinggi, fps, codec video, ukuran file.

## ffmpeg

- Selalu `-hide_banner -nostdin -y`.
- Progres: `-progress pipe:1` menghasilkan baris `kunci=nilai`; persen dihitung dari `out_time_us` dibagi durasi dari ffprobe.
- Ekstrak audio untuk Whisper: mono, 16 kHz, WAV PCM 16-bit (`-vn -ac 1 -ar 16000 -c:a pcm_s16le`).
- Thumbnail: satu frame dari sekitar 10% durasi, lebar 480.
- Konversi ke H.264 (jika sumber bukan H.264): resolusi dan fps tetap, audio AAC. Encoder dipilih dari settings: `libx264` sebagai default, `h264_nvenc` jika GPU dipilih dan tersedia.

## Whisper

- Program yang dipakai adalah whisper.cpp (`whisper-cli`) dengan model ggml dari settings.
- Masukan: `audio.wav` hasil ekstrak. Bahasa: kode dari project, atau `auto`.
- Minta keluaran JSON ke file di folder project, lalu ubah ke format `transcript.json` dan `transcript.txt` milik app (lihat `docs/prd.md`). Konversi ini fungsi murni dengan unit test.
- Progres: aktifkan opsi cetak progres jika ada; jika tidak, hitung dari timestamp segmen terakhir dibagi durasi audio.
- Jumlah thread default: jumlah core fisik, maksimal 8.
- Teks segmen di-trim; segmen kosong dan segmen berulang persis berturut-turut dibuang.
- GPU atau CPU dibaca dari baris awal keluaran, bukan ditebak dari nama file: `loaded CUDA backend` / `using CUDA0 backend` berarti GPU; `no GPU found`, atau hanya `loaded CPU backend`, berarti CPU. Build cuBLAS yang `ggml-cuda.dll`-nya gagal dimuat tetap jalan di CPU tanpa error. "Periksa Tool" membaca baris ini dari `whisper-cli --version` (`parseBackend`, dites dengan `testdata/version_*.txt`).
- Versi diambil dari baris yang memuat "version" (`whisper.cpp version: ...`), karena build CUDA mencetak baris perangkat lebih dulu.

## Tes

- Parser dites dengan contoh keluaran asli yang disimpan di `testdata/`.
- Tes tidak menjalankan program luar. Untuk menguji alur, sediakan skrip palsu kecil di `testdata/` hanya jika benar-benar perlu.
