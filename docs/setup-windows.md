# Setup Windows — SMEditor

App web SMEditor dikembangkan di Ubuntu tapi dipakai sehari-hari di Windows. Dokumen ini menjelaskan langkah demi langkah menyiapkan `smeditor.exe` dan semua tool luar yang dipanggilnya (yt-dlp, ffmpeg/ffprobe, Whisper), sebelum app pertama kali dijalankan.

Lihat juga `docs/erd.md` (tabel `settings`) dan skill `.agents/skills/sm-external-tools` untuk detail cara app memanggil tiap tool.

## 1. Siapkan folder

Pilih satu folder sebagai tempat app berjalan, misalnya `D:\SMEditor\`. Semua langkah di bawah mengasumsikan folder ini. Di dalamnya nanti ada:

```
D:\SMEditor\
  smeditor.exe
  tools\
    yt-dlp.exe
    ffmpeg.exe
    ffprobe.exe
    whisper-cli.exe
    <file DLL whisper.cpp dan CUDA, lihat langkah 4>
    models\
      ggml-medium.bin
  data\                 (dibuat otomatis oleh app saat pertama jalan)
```

`tools\` dan `data\` **tidak** ikut dibuat oleh `make build-windows`; buat sendiri `tools\` (dan `tools\models\`) lalu isi dengan file di bawah. `data\` dibuat otomatis oleh app saat pertama kali jalan (database SQLite dan folder project).

## 2. Unduh yt-dlp

- Sumber: halaman Releases resmi yt-dlp — `https://github.com/yt-dlp/yt-dlp/releases/latest`.
- Unduh aset `yt-dlp.exe` (bukan `yt-dlp` tanpa ekstensi, itu untuk Linux/macOS).
- Taruh di `tools\yt-dlp.exe`.

## 3. Unduh ffmpeg dan ffprobe

- Sumber: halaman build Windows yang dirujuk resmi oleh ffmpeg.org — `https://www.gyan.dev/ffmpeg/builds/`. Unduh paket **"release essentials"** (zip).
- Dari isi zip, ambil `bin\ffmpeg.exe` dan `bin\ffprobe.exe` saja (abaikan `ffplay.exe` dan file lain).
- Taruh keduanya langsung di `tools\ffmpeg.exe` dan `tools\ffprobe.exe` (bukan di subfolder `bin\`).

## 4. Unduh whisper.cpp dengan dukungan GPU NVIDIA

- Sumber: halaman Releases whisper.cpp — `https://github.com/ggml-org/whisper.cpp/releases` (nama organisasi GitHub proyek ini pernah pindah dari `ggerganov/whisper.cpp`; kalau tautan di atas tidak berlaku, cari "whisper.cpp releases" dan pastikan domainnya `github.com`).
- Setiap rilis punya beberapa aset Windows. Cari yang namanya menyebut **CUDA** atau **cuBLAS** (dukungan GPU NVIDIA) dan arsitektur **x64**, bukan aset CPU-only biasa. Nama file berubah antar rilis, jadi cocokkan kata kuncinya, bukan nama file persis.
- Kartu grafis harus NVIDIA dengan CUDA Toolkit/driver yang sesuai versi rilis tersebut terpasang di Windows (lihat catatan rilis whisper.cpp untuk versi CUDA yang dipakai build itu). Tanpa ini, binary CUDA tidak akan jalan meski file sudah ditaruh di tempatnya.
- Dari isi zip, ambil:
  - `whisper-cli.exe` (program yang dipanggil app; versi lama kadang bernama `main.exe` — kalau begitu, ubah nama `whisper_path` di Pengaturan app menjadi `tools/main`, atau ganti nama file itu sendiri jadi `whisper-cli.exe`).
  - Semua file `.dll` yang ikut dalam zip (runtime whisper.cpp dan CUDA) — taruh **sejajar** dengan `whisper-cli.exe` di `tools\`, jangan dipisah ke subfolder, karena Windows mencari DLL di folder yang sama dengan .exe yang memanggilnya.
- Taruh `whisper-cli.exe` di `tools\whisper-cli.exe`, dan semua `.dll` pasangannya juga langsung di `tools\`.

## 5. Unduh model Whisper (ggml-medium)

- Sumber: repo model resmi whisper.cpp di Hugging Face — `https://huggingface.co/ggerganov/whisper.cpp` — unduh file `ggml-medium.bin`.
- Taruh di `tools\models\ggml-medium.bin`.
- Model ini sekitar 1.5 GB; pastikan koneksi stabil atau unduh dari browser (bukan hanya klik kanan "Save Link As" yang kadang memotong file besar).

## 6. Dapatkan smeditor.exe

Dibuild dari Ubuntu (tidak perlu Windows untuk build; cross-compile):

```
make build-windows
```

Hasilnya `backend/bin/smeditor.exe`, sudah menanam frontend (tidak perlu folder `frontend/dist` terpisah di Windows). Salin file ini ke `D:\SMEditor\smeditor.exe`.

## 7. Jalankan dan buka app

1. Klik dua kali `smeditor.exe` (atau jalankan dari Command Prompt/PowerShell di folder yang sama: `.\smeditor.exe`, supaya pesan error terlihat kalau app gagal start — misalnya port terpakai).
2. Sebuah jendela konsol akan muncul dan menampilkan log seperti `smeditor listening on 127.0.0.1:8080 (base dir: D:\SMEditor)`. Biarkan jendela ini terbuka selama app dipakai; menutupnya menghentikan server.
3. Buka browser ke `http://127.0.0.1:8080` (atau `http://localhost:8080`).
4. Windows Firewall mungkin menampilkan prompt izin saat pertama jalan — pilih **Allow** untuk jaringan **Private**. App hanya mendengarkan di `127.0.0.1` jadi tidak diakses dari komputer lain.

## 8. Atur path tool di halaman Pengaturan

Default path beberapa tool (lihat `docs/erd.md`) berasumsi `ffmpeg`/`ffprobe` sudah ada di PATH sistem, yang **tidak** berlaku pada setup mandiri di atas. Setelah app jalan, buka halaman **Pengaturan** dan isi:

| Pengaturan | Nilai |
|---|---|
| Path yt-dlp | `tools/yt-dlp` |
| Path ffmpeg | `tools/ffmpeg` |
| Path ffprobe | `tools/ffprobe` |
| Path Whisper | `tools/whisper-cli` |
| Model Whisper | `tools/models/ggml-medium.bin` |
| Perangkat Whisper | `auto` (boleh pakai GPU jika terdeteksi) atau `gpu`; pilih `cpu` untuk memaksa CPU saja |

Path relatif ini dihitung dari folder `smeditor.exe`, bukan dari folder tempat Command Prompt dibuka — lihat `tools.ResolveExecutable` di `.agents/skills/sm-external-tools`. `.exe` tidak perlu ditulis; app menambahkannya otomatis di Windows.

Klik **Simpan**, lalu klik **Periksa Tool** — keempat tool dan model harus muncul "ditemukan". Kalau salah satu masih "tidak ditemukan", periksa ulang nama file dan lokasinya sesuai tabel di langkah 1.

> Catatan: konversi video ke H.264 (saat sumber bukan H.264) saat ini selalu berjalan di CPU (`libx264`) terlepas dari pengaturan `whisper_device` — GPU di app ini hanya dipakai untuk transcript Whisper, bukan untuk encode video.

## 9. Uji alur dari awal sampai akhir

Sesuai kriteria terima SM-10, pastikan di Windows:

1. Buat project baru dengan URL YouTube video pendek → unduh berhasil (status berubah ke `menunggu_highlight` lewat `transcript`).
2. Transcript berjalan dengan `whisper_device` = `auto` atau `gpu` → log konsol/Task Manager menunjukkan proses GPU terpakai (lihat kolom GPU di Task Manager saat `whisper-cli.exe` berjalan).
3. Tempel JSON highlight contoh dan validasi berhasil → `highlight.json` dan `narasi.txt` tersimpan, status `siap_premiere`.

## Catatan Windows untuk "Buka folder project" dan "Salin ke folder"

Kedua fitur ini punya jalur kode khusus Windows (`backend/internal/tools/open_windows.go`, dan `ExportToFolder`/`ExportRunner` di modul `project` yang memakai `path/filepath` standar Go, bukan path Unix tertulis tetap) dan sudah ikut ter-compile oleh `make build-windows` (`GOOS=windows`). Yang perlu diuji manual di Windows, karena sandbox pengembangan ini tidak punya Windows:

- **Buka folder project**: tombol di halaman project harus membuka Windows Explorer pada folder project (`data\projects\<id>\`), bukan terpotong oleh pembatalan request seperti bug lama di Linux (lihat commit perbaikan `open_windows.go`). Path yang ditampilkan dengan tombol salin harus berupa path Windows dengan backslash dan huruf drive (misalnya `D:\SMEditor\data\projects\01J...\`).
- **Salin ke folder**: isi folder tujuan dengan path Windows (misalnya `E:\Premiere\ProjectX`), termasuk path dengan spasi di namanya. Nama subfolder yang dibuat harus sudah bersih dari karakter yang dilarang Windows (`< > : " / \ | ? *`) — fungsi `SanitizeFolderName` sudah menangani ini lintas OS dan punya unit test, tapi jalur nyata (benar-benar menulis ke disk NTFS, termasuk rename-menimpa file yang sudah ada) hanya bisa dipastikan di Windows asli.
