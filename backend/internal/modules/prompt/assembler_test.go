package prompt

import (
	"regexp"
	"strings"
	"testing"
)

var anyPlaceholder = regexp.MustCompile(`\{\w+\}`)

func mobaGameMode() GameMode {
	return GameMode{
		Code: "mlbb", Name: "Mobile Legends", Genre: "moba", BlockCode: "moba",
		Terms: map[string]string{"objektif": "Turtle, Lord", "bangunan": "Turret, base", "penghargaan": "MVP"},
	}
}

func brGameMode() GameMode {
	return GameMode{
		Code: "pubgm", Name: "PUBG Mobile", Genre: "br", BlockCode: "br",
		Terms: map[string]string{"istilah_menang": "Winner Winner Chicken Dinner"},
	}
}

func fpsGameMode() GameMode {
	return GameMode{
		Code: "valorant", Name: "Valorant", Genre: "fps", BlockCode: "fps",
		Terms: map[string]string{"istilah_karakter": "agent"},
	}
}

func bolaGameMode() GameMode {
	return GameMode{Code: "efootball", Name: "eFootball", Genre: "bola", BlockCode: "bola", Terms: map[string]string{}}
}

func umumGameMode() GameMode {
	return GameMode{Code: "umum", Name: "Umum", Genre: "umum", BlockCode: "umum", Terms: map[string]string{}}
}

func blockFor(t *testing.T, code string) Block {
	t.Helper()
	def, ok := defaultBlocks[code]
	if !ok {
		t.Fatalf("no default block for %q", code)
	}
	return Block{Code: code, Body: def.Body, Categories: def.Categories}
}

func fullInput() AssembleInput {
	return AssembleInput{
		VideoTitle:    "MPL ID S13 Grand Final Game 3",
		DurationSec:   3725, // 01:02:05
		TeamA:         "ONIC",
		TeamB:         "RRQ",
		TargetMinutes: 12,
	}
}

func TestAssemble_OneGamePerGenre(t *testing.T) {
	frame := blockFor(t, "frame")
	cases := []struct {
		name  string
		gm    GameMode
		block string
	}{
		{"moba/mlbb", mobaGameMode(), "moba"},
		{"br/pubgm", brGameMode(), "br"},
		{"fps/valorant", fpsGameMode(), "fps"},
		{"bola/efootball", bolaGameMode(), "bola"},
		{"umum/umum", umumGameMode(), "umum"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			block := blockFor(t, tc.block)
			in := fullInput()
			in.GameCode = tc.gm.Code

			got := assemble(frame, block, tc.gm, in)

			if m := anyPlaceholder.FindString(got); m != "" {
				t.Errorf("assemble() left an unresolved placeholder %q in:\n%s", m, got)
			}
			if !strings.Contains(got, tc.gm.Name) {
				t.Errorf("assemble() missing game name %q", tc.gm.Name)
			}
			if !strings.Contains(got, `"game": "`+tc.gm.Code+`"`) {
				t.Errorf("assemble() missing kode_game %q in format jawaban", tc.gm.Code)
			}
			if !strings.Contains(got, "MPL ID S13 Grand Final Game 3") {
				t.Error("assemble() missing judul")
			}
			if !strings.Contains(got, "01:02:05") {
				t.Error("assemble() missing formatted durasi")
			}
			if !strings.Contains(got, "ONIC vs RRQ") {
				t.Error("assemble() missing tim_a vs tim_b")
			}
			if !strings.Contains(got, "12") {
				t.Error("assemble() missing target_durasi")
			}
			categories := strings.Join(block.Categories, ", ")
			if categories != "" && !strings.Contains(got, categories) {
				t.Errorf("assemble() missing daftar_kategori %q", categories)
			}
		})
	}
}

func TestAssemble_MobaTermsSubstituted(t *testing.T) {
	frame := blockFor(t, "frame")
	block := blockFor(t, "moba")
	gm := mobaGameMode()
	in := fullInput()
	in.GameCode = gm.Code

	got := assemble(frame, block, gm, in)

	for _, term := range []string{"Turtle, Lord", "Turret, base", "MVP"} {
		if !strings.Contains(got, term) {
			t.Errorf("assemble() missing MOBA term %q", term)
		}
	}
}

func TestAssemble_OptionalFieldsDefaultToTidakDiisi(t *testing.T) {
	frame := blockFor(t, "frame")
	block := blockFor(t, "umum")
	gm := umumGameMode()
	in := AssembleInput{GameCode: gm.Code} // everything else empty/zero

	got := assemble(frame, block, gm, in)

	if m := anyPlaceholder.FindString(got); m != "" {
		t.Errorf("assemble() left an unresolved placeholder %q", m)
	}
	if !strings.Contains(got, "Judul video: tidak diisi") {
		t.Error("assemble() should default empty judul to 'tidak diisi'")
	}
	if !strings.Contains(got, "Tim: tidak diisi vs tidak diisi") {
		t.Error("assemble() should default empty tim_a/tim_b to 'tidak diisi'")
	}
	if strings.Contains(got, "Perkiraan total durasi highlight") {
		t.Error("assemble() should drop the whole target_durasi line when TargetMinutes is 0, not fill it with 'tidak diisi'")
	}
}

func TestAssemble_TargetMinutesLineOmittedWhenZero(t *testing.T) {
	frame := blockFor(t, "frame")
	block := blockFor(t, "umum")
	gm := umumGameMode()
	in := fullInput()
	in.GameCode = gm.Code
	in.TargetMinutes = 0

	got := assemble(frame, block, gm, in)

	if m := anyPlaceholder.FindString(got); m != "" {
		t.Errorf("assemble() left an unresolved placeholder %q", m)
	}
	if strings.Contains(got, "Perkiraan total durasi highlight") {
		t.Errorf("assemble() = %q, want the target_durasi line removed entirely when TargetMinutes is 0", got)
	}
}

func TestAssemble_TargetMinutesLineShownWhenSet(t *testing.T) {
	frame := blockFor(t, "frame")
	block := blockFor(t, "umum")
	gm := umumGameMode()
	in := fullInput()
	in.GameCode = gm.Code
	in.TargetMinutes = 15

	got := assemble(frame, block, gm, in)

	if !strings.Contains(got, "Perkiraan total durasi highlight: 15 menit") {
		t.Errorf("assemble() = %q, want the target_durasi line with value 15", got)
	}
}

func TestAssemble_BRBlockTimFokusNeverLeavesPlaceholder(t *testing.T) {
	frame := blockFor(t, "frame")
	block := blockFor(t, "br")
	gm := brGameMode()
	in := fullInput()
	in.GameCode = gm.Code

	got := assemble(frame, block, gm, in)

	if m := anyPlaceholder.FindString(got); m != "" {
		t.Errorf("assemble() left an unresolved placeholder %q", m)
	}
	if !strings.Contains(got, "Winner Winner Chicken Dinner") {
		t.Error("assemble() missing istilah_menang")
	}
}
