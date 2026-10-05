---
name: sm-api-response
description: Format respons JSON, kode error, dan helper httpx di SMEditor, termasuk cara frontend membacanya. Pakai saat menulis handler, menambah jenis error, atau menulis fungsi di frontend/src/api/.
---

# Respons API

Semua endpoint di bawah `/api` mengembalikan JSON dengan salah satu dari dua bentuk. Tidak ada bentuk ketiga.

## Bentuk

Berhasil:

```json
{ "data": { "id": "01J...", "name": "MPL Game 3" } }
```

Gagal:

```json
{ "error": { "code": "invalid_url", "message": "URL harus berupa tautan YouTube" } }
```

Gagal dengan rincian (dipakai validasi highlight):

```json
{
  "error": {
    "code": "highlight_invalid",
    "message": "Ada 2 segmen yang bermasalah",
    "details": [
      { "segmen": 3, "field": "selesai", "message": "Melewati durasi video" }
    ]
  }
}
```

- Daftar selalu berupa array di dalam `data`, tidak pernah `null`. Daftar kosong adalah `[]`.
- `code` berupa snake_case bahasa Inggris dan tidak berubah. `message` berbahasa Indonesia dan boleh ditampilkan langsung ke pengguna.
- Nama field JSON memakai snake_case.
- Waktu berformat RFC 3339 UTC. Durasi dalam detik (angka).

## Status HTTP

| Status | Dipakai untuk |
|---|---|
| 200 | Berhasil membaca atau mengubah |
| 201 | Berhasil membuat |
| 204 | Berhasil menghapus, tanpa body |
| 400 | Body atau parameter tidak bisa dibaca |
| 404 | Data tidak ditemukan |
| 409 | Bentrok dengan keadaan sekarang, misalnya job masih berjalan |
| 422 | Data terbaca tetapi melanggar aturan |
| 500 | Kesalahan tak terduga |

## Helper `internal/httpx`

Handler hanya memakai fungsi ini:

```go
httpx.OK(c, data)        // 200
httpx.Created(c, data)   // 201
httpx.NoContent(c)       // 204
httpx.Fail(c, err)       // memetakan error ke status dan body
```

Error dibuat di service dengan konstruktor:

```go
httpx.ErrBadRequest(code, message)        // 400
httpx.ErrNotFound(code, message)          // 404
httpx.ErrConflict(code, message)          // 409
httpx.ErrUnprocessable(code, message)     // 422
httpx.ErrUnprocessableDetails(code, message, details) // 422 dengan details
```

Semuanya mengembalikan `*httpx.AppError`, yang mengimplementasikan `error`. `httpx.Fail` memakai `errors.As`: jika error adalah `AppError`, status dan body mengikuti isinya. Jika bukan, error dicatat ke log dan klien menerima 500 dengan `code: "internal_error"` dan pesan umum. Rincian error internal tidak pernah dikirim ke klien.

Paket `httpx` tidak mengimpor `gin` di tipe `AppError`, sehingga service boleh memakainya tanpa melanggar aturan lapisan.

## Kode error yang sudah dipakai

| Code | Status | Arti |
|---|---|---|
| `invalid_body` | 400 | Body tidak bisa dibaca |
| `invalid_id` | 400 | ID bukan ULID |
| `invalid_url` | 422 | Bukan URL YouTube |
| `name_required` | 422 | Nama project kosong |
| `game_not_found` | 422 | Kode game tidak dikenal |
| `project_not_found` | 404 | Project tidak ada |
| `job_not_found` | 404 | Job tidak ada |
| `job_running` | 409 | Masih ada job berjalan untuk project ini |
| `transcript_missing` | 409 | Transcript belum ada |
| `highlight_invalid` | 422 | JSON highlight melanggar aturan |
| `tool_not_found` | 409 | yt-dlp, ffmpeg, atau Whisper tidak ditemukan |
| `internal_error` | 500 | Kesalahan tak terduga |

Sebelum membuat kode baru, periksa tabel ini. Jika menambah, tambahkan juga barisnya di sini.

## Pengecualian

- `GET /api/projects/:id/events` adalah SSE (`text/event-stream`). Tiap event berisi JSON: `{"type":"progress|done|failed","job_id":"...","progress":42.5,"message":"..."}`.
- Endpoint unduh file (`transcript`, `narasi`) mengembalikan isi file dengan header `Content-Disposition`. Jika gagal, tetap mengembalikan bentuk error JSON.

## Frontend

Semua panggilan lewat `frontend/src/api/client.js`:

```js
export async function request(method, path, body) {
  const res = await fetch(`/api${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 204) return null
  const json = await res.json()
  if (!res.ok) throw new ApiError(json.error, res.status)
  return json.data
}
```

- `ApiError` membawa `code`, `message`, `details`, dan `status`.
- Komponen menampilkan `err.message` lewat SweetAlert; logika percabangan memakai `err.code`, bukan teks pesan.
- Komponen tidak memanggil `fetch` langsung.
