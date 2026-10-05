# Plugin Premiere - SMEditor Highlight

Panel CEP untuk Premiere Pro 2023 (23.x). Membaca `highlight.json` dari SMEditor dan, pada tahap berikutnya, memotong video asli menjadi sequence.

Status: **tahap 1 (SM-11)** - panel tampil, tersambung ke Premiere, dan membaca `highlight.json`. Pembuatan sequence (SM-12) dan marker (SM-13) belum ada.

## Struktur

```
premiere-plugin/
  CSXS/manifest.xml     identitas panel dan versi Premiere yang didukung
  client/               panel (HTML, CSS, JavaScript modern)
  host/index.jsx        skrip host (ExtendScript) yang berbicara dengan Premiere
```

## Pasang (sekali saja)

Jalankan di PowerShell. Sesuaikan path jika repo tidak berada di `Documents\SMEditor`.

1. Aktifkan mode debug CEP, supaya panel tanpa tanda tangan Adobe boleh dimuat:

   ```powershell
   reg add HKCU\Software\Adobe\CSXS.10 /v PlayerDebugMode /t REG_SZ /d 1 /f
   reg add HKCU\Software\Adobe\CSXS.11 /v PlayerDebugMode /t REG_SZ /d 1 /f
   reg add HKCU\Software\Adobe\CSXS.12 /v PlayerDebugMode /t REG_SZ /d 1 /f
   ```

2. Tautkan folder plugin ke folder ekstensi CEP:

   ```powershell
   New-Item -ItemType Directory -Force "$env:APPDATA\Adobe\CEP\extensions" | Out-Null
   New-Item -ItemType Junction -Path "$env:APPDATA\Adobe\CEP\extensions\com.smeditor.highlight" -Target "$HOME\Documents\SMEditor\premiere-plugin"
   ```

   Karena berupa tautan, perubahan file di repo langsung terpakai tanpa menyalin ulang.

3. Tutup Premiere sepenuhnya, lalu buka lagi.

4. Buka panel lewat menu **Window > Extensions > SMEditor Highlight**.

## Setelah mengubah kode

- Perubahan di `client/`: tutup panel lalu buka lagi dari menu.
- Perubahan di `host/index.jsx` atau `CSXS/manifest.xml`: tutup dan buka ulang Premiere.

## Copot

```powershell
Remove-Item "$env:APPDATA\Adobe\CEP\extensions\com.smeditor.highlight"
```

Perintah ini hanya menghapus tautan, bukan folder di repo.
