package highlight

import "strings"

// TotalDuration sums each segment's own length (selesai - mulai), i.e.
// how long the cut-together highlight reel will actually run — not the
// span from the first segment's start to the last segment's end, which
// would also count the gaps skipped in between. Segments with a time
// that fails to parse are skipped; Validate would have already rejected
// those before this is ever called from Service.
func TotalDuration(segmen []Segment) float64 {
	var total float64
	for _, s := range segmen {
		start, startOK := parseHMS(s.Mulai)
		end, endOK := parseHMS(s.Selesai)
		if startOK && endOK && end > start {
			total += end - start
		}
	}
	return total
}

// buildNarasi renders narasi.txt: one block per segment, in order, with
// its time range and label as a heading and the dubbing script below,
// per docs/prd.md ("Script narasi dibuat dari JSON yang sama ... berurutan
// per segmen").
func buildNarasi(segmen []Segment) string {
	var b strings.Builder
	for _, s := range segmen {
		b.WriteString("[")
		b.WriteString(s.Mulai)
		b.WriteString(" - ")
		b.WriteString(s.Selesai)
		b.WriteString("] ")
		b.WriteString(s.Label)
		b.WriteString("\n")
		b.WriteString(s.Narasi)
		b.WriteString("\n\n")
	}
	return b.String()
}
