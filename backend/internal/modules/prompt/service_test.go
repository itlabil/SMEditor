package prompt

import (
	"context"
	"path/filepath"
	"testing"

	"smeditor/internal/db"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	return NewService(NewRepository(conn))
}

// TestDefaultBlocksMatchSeedMigration guards against defaults.go drifting
// from the seeded blocks after every migration has run (0002 seeds them,
// 0005 replaces the moba block), which can never be edited once committed (per .agents/skills/sm-database) —
// if this test fails, defaults.go has a transcription error, not the
// migration.
func TestDefaultBlocksMatchSeedMigration(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	for code, def := range defaultBlocks {
		seeded, err := svc.GetBlock(ctx, code)
		if err != nil {
			t.Fatalf("GetBlock(%s): %v", code, err)
		}
		if seeded.Body != def.Body {
			t.Errorf("defaultBlocks[%q].Body does not match the seeded migration row:\nseed:\n%s\n\ndefaults.go:\n%s", code, seeded.Body, def.Body)
		}
		if len(seeded.Categories) != len(def.Categories) {
			t.Errorf("defaultBlocks[%q].Categories = %v, want %v (from seed)", code, def.Categories, seeded.Categories)
			continue
		}
		for i := range seeded.Categories {
			if seeded.Categories[i] != def.Categories[i] {
				t.Errorf("defaultBlocks[%q].Categories = %v, want %v (from seed)", code, def.Categories, seeded.Categories)
				break
			}
		}
	}
}

func TestServiceListGameModes_ReturnsSeedData(t *testing.T) {
	svc := newTestService(t)

	list, err := svc.ListGameModes(context.Background())
	if err != nil {
		t.Fatalf("ListGameModes: %v", err)
	}
	if len(list) != 13 { // 12 games + umum, per docs/prd.md
		t.Fatalf("len(list) = %d, want 13", len(list))
	}

	byCode := map[string]GameMode{}
	for _, gm := range list {
		byCode[gm.Code] = gm
	}
	mlbb, ok := byCode["mlbb"]
	if !ok {
		t.Fatal("mlbb not found in game modes")
	}
	if mlbb.Terms["objektif"] != "Turtle, Lord" {
		t.Errorf("mlbb terms[objektif] = %q, want Turtle, Lord", mlbb.Terms["objektif"])
	}
}

func TestServiceUpdateBlock_PersistsAndMarksCustom(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	got, err := svc.UpdateBlock(ctx, "umum", UpdateBlockRequest{
		Body:       "Isi baru untuk blok umum",
		Categories: []string{"a", "b"},
	})
	if err != nil {
		t.Fatalf("UpdateBlock: %v", err)
	}
	if got.Body != "Isi baru untuk blok umum" || !got.IsCustom {
		t.Errorf("UpdateBlock result = %+v, want updated body and is_custom=true", got)
	}

	again, err := svc.GetBlock(ctx, "umum")
	if err != nil {
		t.Fatalf("GetBlock: %v", err)
	}
	if again.Body != "Isi baru untuk blok umum" {
		t.Errorf("GetBlock after update = %q, want it persisted", again.Body)
	}
}

func TestServiceUpdateBlock_RejectsEmptyBody(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.UpdateBlock(context.Background(), "umum", UpdateBlockRequest{Body: "   "})
	if err == nil {
		t.Fatal("UpdateBlock with blank body: want error, got nil")
	}
}

func TestServiceUpdateBlock_UnknownCodeNotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.UpdateBlock(context.Background(), "not-a-real-block", UpdateBlockRequest{Body: "x"})
	if err == nil {
		t.Fatal("UpdateBlock with unknown code: want error, got nil")
	}
}

func TestServiceResetBlock_RestoresSeedAfterEdit(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	original, err := svc.GetBlock(ctx, "umum")
	if err != nil {
		t.Fatalf("GetBlock: %v", err)
	}

	if _, err := svc.UpdateBlock(ctx, "umum", UpdateBlockRequest{Body: "diubah", Categories: []string{"x"}}); err != nil {
		t.Fatalf("UpdateBlock: %v", err)
	}

	reset, err := svc.ResetBlock(ctx, "umum")
	if err != nil {
		t.Fatalf("ResetBlock: %v", err)
	}
	if reset.Body != original.Body {
		t.Errorf("ResetBlock().Body = %q, want original %q", reset.Body, original.Body)
	}
	if reset.IsCustom {
		t.Error("ResetBlock() should clear is_custom")
	}
}

func TestServiceAssemble_UnknownGameCode(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Assemble(context.Background(), AssembleInput{GameCode: "not-a-real-game"})
	if err == nil {
		t.Fatal("Assemble with unknown game code: want error, got nil")
	}
}

func TestServiceAssemble_RealGameFromDB(t *testing.T) {
	svc := newTestService(t)

	text, err := svc.Assemble(context.Background(), AssembleInput{
		GameCode:   "valorant",
		VideoTitle: "Grand Final",
		TeamA:      "T1",
		TeamB:      "DRX",
	})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if m := anyPlaceholder.FindString(text); m != "" {
		t.Errorf("Assemble() left an unresolved placeholder %q", m)
	}
}

func TestServiceCategoriesForGame(t *testing.T) {
	svc := newTestService(t)

	got, err := svc.CategoriesForGame(context.Background(), "mlbb")
	if err != nil {
		t.Fatalf("CategoriesForGame: %v", err)
	}
	want := []string{"draft", "early", "mid", "end", "kesimpulan"}
	if len(got) != len(want) {
		t.Fatalf("got = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestServiceCategoriesForGame_UnknownGame(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.CategoriesForGame(context.Background(), "not-a-real-game")
	if err == nil {
		t.Fatal("CategoriesForGame with unknown code: want error, got nil")
	}
}
