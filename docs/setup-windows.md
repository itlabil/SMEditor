# Setup Windows — SMEditor

App web SMEditor bisa dibuild langsung di Windows (`build.ps1`) atau di-cross-compile dari Ubuntu, dan dipakai sehari-hari di Windows. Dokumen ini menjelaskan langkah demi langkah menyiapkan `smeditor.exe` dan semua tool luar yang dipanggilnya (yt-dlp, ffmpeg/ffprobe, Whisper), sebelum app pertama kali dijalankan.

Lihat juga `docs/erd.md` (tabel `settings`) dan skill `.agents/skills/sm-external-tools` untuk detail cara app memanggil tiap tool.

## 1. Siapkan folder

Pilih satu folder sebagai tempat app berjalan. Jika build dilakukan di Windows dengan `build.ps1` (langkah 6), folder ini adalah root repo itu sendiri, karena `smeditor.exe` dihasilkan di sana. Jika memakai hasil cross-compile dari Ubuntu, pakai folder lain, misalnya `D:\SMEditor\`. Di dalamnya nanti ada:

```
<folder app>\
  smeditor.exe
  tools\
    yt-dlp.exe
    whisper\
      whisper-cli.exe
      whisper.dll, ggml.dll, ggml-base.dll, ggml-cpu-*.dll, SDL2.dll, ...
      (build GPU: ggml-cuda.dll dan DLL runtime CUDA)
    models\
      ggml-medium.bin
  data\                 (dibuat otomatis oleh app saat pertama jalan)
```

ffmpeg dan ffprobe tidak ditaruh di `tools\`; keduanya dipasang lewat winget dan dipanggil dari PATH sistem (langkah 3).

`tools\` dan `data\` **tidak** ikut dibuat oleh build; buat sendiri `tools\`, `tools\whisper\`, dan `tools\models\`, lalu isi dengan file di bawah. `data\` dibuat otomatis oleh app saat pertama kali jalan (database SQLite dan folder project).

## 2. Unduh yt-dlp

- Sumber: halaman Releases resmi yt-dlp — `https://github.com/yt-dlp/yt-dlp/releases/latest`.
- Unduh aset `yt-dlp.exe` (bukan `yt-dlp` tanpa ekstensi, itu untuk Linux/macOS).
- Taruh di `tools\yt-dlp.exe`.

## 3. Pasang ffmpeg dan ffprobe lewat winget

Jalankan di PowerShell atau Command Prompt:

```
winget install Gyan.FFmpeg
```

Paket ini memasang `ffmpeg.exe` dan `ffprobe.exe` sekaligus dan menambahkannya ke PATH pengguna. **Tutup lalu buka ulang** jendela terminal (dan `smeditor.exe` jika sedang jalan) supaya PATH baru terbaca. Periksa dengan:

```
ffmpeg -version
ffprobe -version
```

Karena keduanya ada di PATH, nilai bawaan pengaturan (`ffmpeg` dan `ffprobe`) sudah benar dan tidak perlu diubah.

## 4. Unduh whisper.cpp

- Sumber: halaman Releases whisper.cpp — `https://github.com/ggml-org/whisper.cpp/releases`.
- **Rilis bertag versi seperti `v1.9.4` tidak punya file unduhan (aset) binary.** Binary Windows ada di rilis build bernomor, yaitu rilis **`b5130`**. Buka rilis itu, lalu bagian *Assets*.
- Pilih salah satu:
  - `whisper-bin-x64.zip` — CPU saja. Jalan di semua PC, tanpa syarat driver.
  - `whisper-cublas-*-bin-x64.zip` — GPU NVIDIA (CUDA/cuBLAS); `*` adalah versi CUDA build itu. Butuh kartu NVIDIA dengan driver yang mendukung versi CUDA tersebut. Pakai ini supaya transcript bisa berjalan di GPU.
- Ekstrak **seluruh isi** zip, termasuk semua file `.dll`, ke `tools\whisper\`. Jika zip berisi subfolder (misalnya `Release\`), pindahkan isinya supaya `whisper-cli.exe` berada langsung di `tools\whisper\whisper-cli.exe`.
- Semua `.dll` (`whisper.dll`, `ggml.dll`, `ggml-base.dll`, `ggml-cpu-*.dll`, `SDL2.dll`, dan untuk build GPU `ggml-cuda.dll` beserta DLL runtime CUDA) wajib berada **sejajar** dengan `whisper-cli.exe` di `tools\whisper\`. Windows mencari DLL di folder yang sama dengan .exe yang memanggilnya; jika ada DLL yang tertinggal, `whisper-cli.exe` gagal start tanpa pesan yang jelas.
- Zip juga berisi banyak program lain (`bench.exe`, `stream.exe`, `main.exe`, dan sebagainya). Boleh dibiarkan; app hanya memanggil `whisper-cli.exe`.

## 5. Unduh model Whisper (ggml-medium)

- Sumber: repo model resmi whisper.cpp di Hugging Face — `https://huggingface.co/ggerganov/whisper.cpp` — unduh file `ggml-medium.bin`.
- Taruh di `tools\models\ggml-medium.bin`.
- Model ini sekitar 1.5 GB; pastikan koneksi stabil atau unduh dari browser (bukan hanya klik kanan "Save Link As" yang kadang memotong file besar).
- Model lain (misalnya `ggml-small.bin`, lebih cepat tapi kurang akurat) boleh dipakai; sesuaikan pengaturan **Model Whisper**.

## 6. Dapatkan smeditor.exe

### Build di Windows

Butuh Go dan Node.js (npm). Dari root repo, jalankan di PowerShell:

```
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

Skrip ini menjalankan `npm install` dan `npm run build` di `frontend\`, menyalin `frontend\dist` ke `backend\internal\webdist\dist`, lalu `go build -tags embed_prod` yang menghasilkan `smeditor.exe` di root repo dengan frontend tertanam.

Jika `smeditor.exe` sedang berjalan, skrip memindahkannya ke `smeditor.exe.old` (Windows mengizinkan mengganti nama .exe yang sedang jalan, tapi tidak menimpanya) lalu menulis `smeditor.exe` baru. App lama tetap berjalan sampai ditutup; **tutup lalu jalankan ulang `smeditor.exe`** supaya versi baru yang dipakai.

### Cross-compile dari Ubuntu

```
make build-windows
```

Hasilnya `backend/bin/smeditor.exe`, sudah menanam frontend (tidak perlu folder `frontend/dist` terpisah di Windows). Salin file ini ke folder app, misalnya `D:\SMEditor\smeditor.exe`.

## 7. Jalankan dan buka app

1. Klik dua kali `smeditor.exe` (atau jalankan dari Command Prompt/PowerShell di folder yang sama: `.\smeditor.exe`, supaya pesan error terlihat kalau app gagal start — misalnya port terpakai).
2. Sebuah jendela konsol akan muncul dan menampilkan log seperti `smeditor listening on 127.0.0.1:8080 (base dir: D:\SMEditor)`. Biarkan jendela ini terbuka selama app dipakai; menutupnya menghentikan server.
3. Buka browser ke `http://127.0.0.1:8080` (atau `http://localhost:8080`).
4. Windows Firewall mungkin menampilkan prompt izin saat pertama jalan — pilih **Allow** untuk jaringan **Private**. App hanya mendengarkan di `127.0.0.1` jadi tidak diakses dari komputer lain.

## 8. Atur path tool di halaman Pengaturan

Default `ffmpeg`/`ffprobe` (lihat `docs/erd.md`) sudah cocok karena keduanya dipasang lewat winget ke PATH. Default Whisper (`tools/whisper-cli`) **tidak** cocok dengan letak di langkah 4, jadi setelah app jalan, buka halaman **Pengaturan** dan isi:

| Pengaturan | Nilai |
|---|---|
| Path yt-dlp | `tools/yt-dlp` |
| Path ffmpeg | `ffmpeg` (dari PATH) |
| Path ffprobe | `ffprobe` (dari PATH) |
| Path Whisper | `tools/whisper/whisper-cli` |
| Model Whisper | `tools/models/ggml-medium.bin` |
| Perangkat Whisper | `auto` (boleh pakai GPU jika terdeteksi) atau `gpu`; pilih `cpu` untuk memaksa CPU saja |

Path relatif ini dihitung dari folder `smeditor.exe`, bukan dari folder tempat Command Prompt dibuka — lihat `tools.ResolveExecutable` di `.agents/skills/sm-external-tools`. `.exe` tidak perlu ditulis; app menambahkannya otomatis di Windows.

Klik **Simpan**, lalu klik **Periksa Tool** — keempat tool dan model harus muncul "ditemukan". Kalau salah satu masih "tidak ditemukan", periksa ulang nama file dan lokasinya sesuai langkah 1. Jika ffmpeg/ffprobe "tidak ditemukan" padahal sudah dipasang, tutup lalu jalankan ulang `smeditor.exe` supaya PATH baru terbaca.

> Catatan: konversi video ke H.264 (saat sumber bukan H.264) saat ini selalu berjalan di CPU (`libx264`) terlepas dari pengaturan `whisper_device` — GPU di app ini hanya dipakai untuk transcript Whisper, bukan untuk encode video.

## 9. Uji alur dari awal sampai akhir

Sesuai kriteria terima SM-10, pastikan di Windows:

1. Buat project baru dengan URL YouTube video pendek → unduh berhasil (status berubah ke `menunggu_highlight` lewat `transcript`).
2. Transcript berjalan dengan `whisper_device` = `auto` atau `gpu` → log konsol/Task Manager menunjukkan proses GPU terpakai (lihat kolom GPU di Task Manager saat `whisper-cli.exe` berjalan).
3. Tempel JSON highlight contoh dan validasi berhasil → `highlight.json` dan `narasi.txt` tersimpan, status `siap_premiere`.

## Catatan Windows untuk "Buka folder project" dan "Salin ke folder"

Kedua fitur ini punya jalur kode khusus Windows (`backend/internal/tools/open_windows.go`, dan `ExportToFolder`/`ExportRunner` di modul `project` yang memakai `path/filepath` standar Go, bukan path Unix tertulis tetap) dan sudah ikut ter-compile oleh `build.ps1` maupun `make build-windows` (`GOOS=windows`). Yang perlu diuji manual di Windows, karena sandbox pengembangan ini tidak punya Windows:

- **Buka folder project**: tombol di halaman project harus membuka Windows Explorer pada folder project (`data\projects\<id>\`), bukan terpotong oleh pembatalan request seperti bug lama di Linux (lihat commit perbaikan `open_windows.go`). Path yang ditampilkan dengan tombol salin harus berupa path Windows dengan backslash dan huruf drive (misalnya `D:\SMEditor\data\projects\01J...\`).
- **Salin ke folder**: isi folder tujuan dengan path Windows (misalnya `E:\Premiere\ProjectX`), termasuk path dengan spasi di namanya. Nama subfolder yang dibuat harus sudah bersih dari karakter yang dilarang Windows (`< > : " / \ | ? *`) — fungsi `SanitizeFolderName` sudah menangani ini lintas OS dan punya unit test, tapi jalur nyata (benar-benar menulis ke disk NTFS, termasuk rename-menimpa file yang sudah ada) hanya bisa dipastikan di Windows asli.
