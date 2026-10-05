package settings

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) GetAll(ctx context.Context) (map[string]string, error) {
	const q = `SELECT key, value FROM settings`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan settings: %w", err)
		}
		out[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows settings: %w", err)
	}
	return out, nil
}

func (r *Repository) Upsert(ctx context.Context, values map[string]string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const q = `INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	for key, value := range values {
		if _, err := tx.ExecContext(ctx, q, key, value); err != nil {
			return fmt.Errorf("upsert setting %s: %w", key, err)
		}
	}

	return tx.Commit()
}
