package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"regexp"
	"testing"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := Migrate(context.Background(), conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return conn
}

func TestMigrateCreatesTables(t *testing.T) {
	conn := openTestDB(t)

	tables := []string{"schema_migrations", "prompt_blocks", "game_modes", "projects", "jobs", "settings"}
	for _, table := range tables {
		var name string
		err := conn.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	conn, err := Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(ctx, conn); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	var count int
	if err := conn.QueryRow(`SELECT count(*) FROM game_modes`).Scan(&count); err != nil {
		t.Fatalf("count game_modes: %v", err)
	}
	if count != 13 {
		t.Errorf("game_modes count = %d after running migrations twice, want 13 (no duplicate seed rows)", count)
	}
}

func TestSeedGameModeCount(t *testing.T) {
	conn := openTestDB(t)

	var count int
	if err := conn.QueryRow(`SELECT count(*) FROM game_modes`).Scan(&count); err != nil {
		t.Fatalf("count game_modes: %v", err)
	}
	if count != 13 {
		t.Errorf("game_modes count = %d, want 13 (12 game + 1 umum)", count)
	}

	var umumCount int
	if err := conn.QueryRow(`SELECT count(*) FROM game_modes WHERE genre = 'umum'`).Scan(&umumCount); err != nil {
		t.Fatalf("count umum game_modes: %v", err)
	}
	if umumCount != 1 {
		t.Errorf("genre=umum count = %d, want 1", umumCount)
	}
}

func TestSeedPromptBlockCount(t *testing.T) {
	conn := openTestDB(t)

	var count int
	if err := conn.QueryRow(`SELECT count(*) FROM prompt_blocks`).Scan(&count); err != nil {
		t.Fatalf("count prompt_blocks: %v", err)
	}
	if count != 6 {
		t.Errorf("prompt_blocks count = %d, want 6 (frame + moba, br, fps, bola, umum)", count)
	}
}

// projectDataPlaceholders lists placeholders filled from project/request data
// at prompt-assembly time (SM-08), not from a game mode's terms_json.
var projectDataPlaceholders = map[string]bool{
	"nama_game":       true,
	"judul":           true,
	"durasi":          true,
	"tim_a":           true,
	"tim_b":           true,
	"target_durasi":   true,
	"blok_tugas":      true,
	"kode_game":       true,
	"daftar_kategori": true,
	"tim_fokus":       true,
}

var placeholderPattern = regexp.MustCompile(`\{(\w+)\}`)

func extractPlaceholders(body string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(body, -1)
	seen := map[string]bool{}
	var out []string
	for _, m := range matches {
		ph := m[1]
		if !seen[ph] {
			seen[ph] = true
			out = append(out, ph)
		}
	}
	return out
}

// TestPromptBlockPlaceholdersHaveValues checks that every {placeholder} in a
// block's body is either filled from the game mode's terms_json or from
// project data, so the assembled prompt in SM-08 never leaves a {...} unfilled.
func TestPromptBlockPlaceholdersHaveValues(t *testing.T) {
	conn := openTestDB(t)

	bodies := map[string]string{}
	rows, err := conn.Query(`SELECT code, body FROM prompt_blocks`)
	if err != nil {
		t.Fatalf("query prompt_blocks: %v", err)
	}
	for rows.Next() {
		var code, body string
		if err := rows.Scan(&code, &body); err != nil {
			t.Fatalf("scan prompt_blocks: %v", err)
		}
		bodies[code] = body
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows prompt_blocks: %v", err)
	}
	rows.Close()

	for _, ph := range extractPlaceholders(bodies["frame"]) {
		if !projectDataPlaceholders[ph] {
			t.Errorf("frame placeholder {%s} is not project data and frame has no terms_json to fill it", ph)
		}
	}

	type gameRow struct {
		code, blockCode, termsJSON string
	}
	var games []gameRow
	rows, err = conn.Query(`SELECT code, block_code, terms_json FROM game_modes`)
	if err != nil {
		t.Fatalf("query game_modes: %v", err)
	}
	for rows.Next() {
		var g gameRow
		if err := rows.Scan(&g.code, &g.blockCode, &g.termsJSON); err != nil {
			t.Fatalf("scan game_modes: %v", err)
		}
		games = append(games, g)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows game_modes: %v", err)
	}
	rows.Close()

	for _, g := range games {
		body, ok := bodies[g.blockCode]
		if !ok {
			t.Errorf("game %s references unknown block_code %s", g.code, g.blockCode)
			continue
		}

		var terms map[string]string
		if err := json.Unmarshal([]byte(g.termsJSON), &terms); err != nil {
			t.Errorf("game %s: invalid terms_json: %v", g.code, err)
			continue
		}

		for _, ph := range extractPlaceholders(body) {
			if projectDataPlaceholders[ph] {
				continue
			}
			if _, ok := terms[ph]; !ok {
				t.Errorf("game %s: placeholder {%s} in block %s has no value in terms_json and is not project data", g.code, ph, g.blockCode)
			}
		}
	}
}
