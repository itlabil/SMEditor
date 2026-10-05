# Arsitektur

Baca `docs/prd.md`, `docs/flow.md`, `docs/erd.md`, `docs/sequence.md` sebelum mengubah perilaku. Jika dokumen dan kode berbeda, ikuti kode yang ada dan laporkan perbedaannya.

## Struktur repo

```
backend/
  cmd/smeditor/main.go
  internal/
    app/            wiring: config, router, start worker
    db/             koneksi, migrasi, seed
    httpx/          helper respons dan error
    modules/
      project/      handler.go, service.go, repository.go, model.go
      job/          handler.go, service.go, repository.go, worker.go, sse.go
      prompt/       handler.go, service.go, repository.go
      highlight/    handler.go, service.go, validator.go
      settings/     handler.go, service.go, repository.go
    tools/
      ytdlp/        pemanggil dan parser progres
      ffmpeg/       ffmpeg dan ffprobe
      whisper/      pemanggil dan konversi keluaran
    storage/        path folder project, baca tulis file
    webdist/        embed frontend/dist untuk mode produksi (build tag embed_prod)
frontend/
  src/
    api/            satu file per modul backend
    pages/
    components/
    composables/    useSSE, useConfirm, dsb.
premiere-plugin/
  CSXS/manifest.xml
  client/           panel HTML/JS
  host/             ExtendScript (.jsx)
docs/
.agents/
```

## Aturan lapisan

- `handler`: hanya membaca request, memanggil service, menulis respons. Tanpa SQL, tanpa logika bisnis.
- `service`: logika bisnis. Tanpa tipe `gin.*` atau `net/http`.
- `repository`: satu-satunya tempat SQL. Tanpa tipe HTTP.
- `validator`: fungsi murni tanpa akses DB atau file, supaya mudah dites.
- `tools/*`: satu-satunya tempat `exec.Command`. Modul lain memanggil lewat interface.
- `storage`: satu-satunya tempat yang menyusun path folder project.
- Antar modul berkomunikasi lewat service, bukan langsung ke repository modul lain.

## Aturan tetap

- Server hanya bind ke `127.0.0.1`.
- Tidak ada panggilan ke API AI. AI selalu di luar app.
- Segmen highlight tidak masuk database; sumbernya `highlight.json`.
- Plugin Premiere tidak berkomunikasi dengan app web; penghubungnya hanya file.
