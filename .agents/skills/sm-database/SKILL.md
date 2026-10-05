---
name: sm-database
description: Cara memakai SQLite di SMEditor - koneksi, migrasi, seed, dan pola query di repository. Pakai saat menambah atau mengubah tabel, menulis migrasi, atau menulis SQL di repository.go.
---

# Database

SMEditor memakai SQLite dalam satu file, `data/app.db`. Skema lengkap ada di `docs/erd.md`; jika skema berubah, dokumen itu ikut diperbarui.

## Driver dan koneksi

Driver wajib `modernc.org/sqlite` (nama driver `sqlite`). Driver ini Go murni, sehingga build ke Windows dari Ubuntu tidak butuh CGO. Jangan memakai `mattn/go-sqlite3`.

```go
dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
db, err := sql.Open("sqlite", dsn)
db.SetMaxOpenConns(1)
```

- `foreign_keys(1)` wajib; tanpa itu `ON DELETE CASCADE` tidak berjalan.
- `SetMaxOpenConns(1)` mencegah error "database is locked" saat worker dan handler menulis bersamaan. Beban app ini kecil, jadi satu koneksi cukup.
- Koneksi dibuat sekali di `internal/app` dan disuntikkan ke repository. Tidak ada variabel `db` global.

## Migrasi

File SQL bernomor di `backend/internal/db/migrations/`, ditanam dengan `go:embed`:

```
0001_init.sql
0002_seed_prompts.sql
```

Aturan:

- Migrasi dijalankan otomatis saat app start, berurutan, masing-masing dalam satu transaksi.
- Versi yang sudah jalan dicatat di tabel `schema_migrations(version, applied_at)`.
- File migrasi yang sudah di-commit tidak pernah diubah. Perubahan skema selalu berupa file baru.
- Tidak ada migrasi turun. Untuk membatalkan, tulis migrasi baru.
- Seed data bawaan (mode game, blok prompt) ditulis sebagai migrasi dengan `INSERT OR IGNORE`, supaya tidak menimpa perubahan pengguna.

## Tipe kolom

| Data | Tipe SQLite | Catatan |
|---|---|---|
| ID | `TEXT PRIMARY KEY` | ULID, dibuat di Go |
| Waktu | `TEXT` | RFC 3339 UTC, ditulis dari Go; jangan memakai `CURRENT_TIMESTAMP` |
| Boolean | `INTEGER` | 0 atau 1 |
| Durasi, fps | `REAL` | Detik |
| JSON | `TEXT` | Di-marshal di Go; nama kolom berakhiran `_json` |
| Status | `TEXT` | Nilainya konstanta di `model.go` |

Kolom opsional memakai `NOT NULL DEFAULT ''` atau `DEFAULT 0` bila memungkinkan, supaya tidak perlu `sql.NullString`. Pakai `NULL` hanya jika "tidak ada" berbeda arti dari "kosong" (misalnya `started_at`).

## Pola query

```go
func (r *Repository) FindByID(ctx context.Context, id string) (*Project, error) {
    const q = `SELECT id, name, status, created_at FROM projects WHERE id = ?`
    var p Project
    var createdAt string
    err := r.db.QueryRowContext(ctx, q, id).Scan(&p.ID, &p.Name, &p.Status, &createdAt)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("find project %s: %w", id, err)
    }
    p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
    return &p, nil
}
```

- Selalu pakai placeholder `?`. Nilai tidak pernah disambung ke string SQL.
- Selalu pakai varian `...Context`.
- Sebutkan nama kolom; jangan `SELECT *`.
- Repository mengembalikan error sentinel modul (`ErrNotFound`) untuk "tidak ada baris"; service yang mengubahnya menjadi `httpx.ErrNotFound`.
- `rows` selalu ditutup dengan `defer rows.Close()` dan `rows.Err()` diperiksa setelah loop.
- Daftar kosong dikembalikan sebagai slice kosong (`[]Project{}`), bukan `nil`.
- SQL pendek ditulis sebagai `const` di dalam fungsi. SQL panjang (lebih dari sekitar 15 baris) dipindah ke `queries.go` di modul yang sama.

## Transaksi

Dipakai jika satu aksi menulis lebih dari satu tabel, misalnya membuat project sekaligus job pertamanya:

```go
tx, err := r.db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()
// ... beberapa tx.ExecContext
return tx.Commit()
```

Operasi file (membuat atau menghapus folder project) tidak bisa di-rollback. Urutannya: saat membuat, tulis DB dulu lalu buat folder; saat menghapus, hapus folder dulu lalu baris DB. Dengan begitu kegagalan di tengah tidak meninggalkan file tanpa pemilik.

## Tes

Tes repository memakai database sementara di `t.TempDir()` dengan migrasi yang sama, bukan mock.
