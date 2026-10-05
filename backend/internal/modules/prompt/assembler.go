package prompt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// placeholderPattern matches {word} tokens only when the whole span
// between braces is word characters, so the literal JSON braces inside
// the frame's "Format jawaban" example (which always contain quotes,
// colons, or newlines) are never mistaken for a placeholder.
var placeholderPattern = regexp.MustCompile(`\{(\w+)\}`)

// assemble fills in frame's placeholders with block's (itself filled in
// first) and in's project data, per docs/prd.md ("Kerangka bersama").
// Every {placeholder} is guaranteed to be replaced by something (falling
// back to an empty string for a name nothing supplies), so the result
// never contains a literal {...} token, per SM-08's acceptance criteria.
func assemble(frame, block Block, gm GameMode, in AssembleInput) string {
	vars := map[string]string{
		"tim_fokus": "", // no dedicated project field; always left for the user to fill in by hand
	}
	for k, v := range gm.Terms {
		vars[k] = v
	}

	blockText := substitute(block.Body, vars)

	vars["nama_game"] = gm.Name
	vars["kode_game"] = gm.Code
	vars["judul"] = orDefault(in.VideoTitle)
	vars["durasi"] = formatHMS(in.DurationSec)
	vars["tim_a"] = orDefault(in.TeamA)
	vars["tim_b"] = orDefault(in.TeamB)
	vars["blok_tugas"] = blockText
	vars["daftar_kategori"] = strings.Join(block.Categories, ", ")

	// target_durasi is optional: PRD asks for the whole line to disappear
	// when it's not set, not just the placeholder (unlike every other
	// optional field, which falls back to "tidak diisi" in place).
	frameBody := frame.Body
	if in.TargetMinutes > 0 {
		vars["target_durasi"] = strconv.Itoa(in.TargetMinutes)
	} else {
		frameBody = removeLineContaining(frameBody, "{target_durasi}")
	}

	return substitute(frameBody, vars)
}

// removeLineContaining drops every line of text that contains token,
// joining what remains back with "\n".
func removeLineContaining(text, token string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, token) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func substitute(text string, vars map[string]string) string {
	return placeholderPattern.ReplaceAllStringFunc(text, func(token string) string {
		key := token[1 : len(token)-1]
		return vars[key] // zero value "" for an unknown key, never the literal token
	})
}

func orDefault(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "tidak diisi"
	}
	return s
}

func formatHMS(sec float64) string {
	total := int(sec)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
