---
name: sm-module-structure
description: Bentuk satu modul backend SMEditor (handler, service, repository, model) dan cara mendaftarkannya. Pakai saat membuat modul baru atau menambah endpoint di backend/internal/modules/.
---

# Struktur modul backend

Setiap fitur backend adalah satu modul di `backend/internal/modules/<nama>/`. Modul selalu berisi file yang sama, sehingga letak kode bisa ditebak tanpa membaca seluruh project.

## File per modul

| File | Isi | Tidak boleh berisi |
|---|---|---|
| `model.go` | Struct domain, struct request dan response, konstanta status | Logika |
| `repository.go` | Semua SQL modul ini | Tipe `gin.*` atau `net/http`, logika bisnis |
| `service.go` | Logika bisnis, memanggil repository dan service lain | SQL, tipe `gin.*` |
| `handler.go` | Bind request, panggil service, tulis respons, daftar route | SQL, logika bisnis, `if` validasi aturan bisnis |
| `validator.go` (opsional) | Fungsi murni untuk validasi yang punya banyak aturan | Akses DB atau file |
| `*_test.go` | Tes untuk service dan validator | |

Modul yang tidak punya tabel sendiri (misalnya `highlight`) boleh tanpa `repository.go`.

## Kerangka

```go
// model.go
package project

type Project struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}

type CreateRequest struct {
    Name       string `json:"name"`
    GameCode   string `json:"game_code"`
    YoutubeURL string `json:"youtube_url"`
}
```

```go
// repository.go
package project

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindByID(ctx context.Context, id string) (*Project, error) {
    // SQL hanya di sini
}
```

```go
// service.go
package project

type Service struct {
    repo    *Repository
    storage *storage.Storage
    jobs    JobEnqueuer // interface, didefinisikan di modul ini
}

type JobEnqueuer interface {
    Enqueue(ctx context.Context, projectID, jobType string) error
}

func NewService(repo *Repository, st *storage.Storage, jobs JobEnqueuer) *Service {
    return &Service{repo: repo, storage: st, jobs: jobs}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Project, error) {
    // validasi aturan bisnis, simpan, buat folder, antrekan job
}
```

```go
// handler.go
package project

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(r *gin.RouterGroup) {
    g := r.Group("/projects")
    g.GET("", h.list)
    g.POST("", h.create)
    g.GET("/:id", h.get)
    g.DELETE("/:id", h.delete)
}

func (h *Handler) create(c *gin.Context) {
    var req CreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httpx.Fail(c, httpx.ErrBadRequest("invalid_body", "Data yang dikirim tidak valid"))
        return
    }
    p, err := h.svc.Create(c.Request.Context(), req)
    if err != nil {
        httpx.Fail(c, err)
        return
    }
    httpx.Created(c, p)
}
```

## Aturan

- Setiap method repository dan service menerima `context.Context` sebagai parameter pertama.
- Modul A yang butuh modul B mendefinisikan interface kecil di modul A (seperti `JobEnqueuer`), lalu service B disuntikkan di `internal/app`. Modul tidak mengimpor repository modul lain.
- Proses luar (yt-dlp, ffmpeg, Whisper) hanya dipanggil lewat paket `internal/tools/*`.
- Path folder project hanya disusun oleh `internal/storage`.
- Handler tidak pernah menulis `c.JSON` langsung; selalu lewat `httpx` (lihat skill `sm-api-response`).

## Mendaftarkan modul

Semua wiring ada di `backend/internal/app/app.go`, berurutan: repository, service, handler, lalu `handler.Register(api)`. Tidak ada variabel global dan tidak ada `init()` yang membuat koneksi.

## Daftar periksa modul baru

1. Buat folder dan file sesuai tabel di atas.
2. Tambahkan migrasi jika butuh tabel (skill `sm-database`).
3. Daftarkan di `app.go`.
4. Tambahkan endpoint ke `docs/sequence.md`.
5. Tulis tes untuk service dan validator.
