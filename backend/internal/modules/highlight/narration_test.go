package highlight

import "testing"

func TestBuildNarasi(t *testing.T) {
	segmen := []Segment{
		{Mulai: "00:01:05", Selesai: "00:02:10", Label: "Draft pick kedua tim", Narasi: "Tim A mengamankan hero incaran di pick pertama"},
		{Mulai: "00:05:00", Selesai: "00:06:00", Label: "Early game", Narasi: "Tim B mendapat first blood"},
	}
	want := "[00:01:05 - 00:02:10] Draft pick kedua tim\n" +
		"Tim A mengamankan hero incaran di pick pertama\n\n" +
		"[00:05:00 - 00:06:00] Early game\n" +
		"Tim B mendapat first blood\n\n"

	if got := buildNarasi(nil, segmen); got != want {
		t.Errorf("buildNarasi() = %q, want %q", got, want)
	}
}

func TestTotalDuration(t *testing.T) {
	cases := []struct {
		name   string
		segmen []Segment
		want   float64
	}{
		{
			name: "two segments",
			segmen: []Segment{
				{Mulai: "00:01:00", Selesai: "00:02:00"}, // 60s
				{Mulai: "00:05:00", Selesai: "00:05:30"}, // 30s
			},
			want: 90,
		},
		{
			name:   "empty",
			segmen: nil,
			want:   0,
		},
		{
			name: "skips a segment with an unparsable time",
			segmen: []Segment{
				{Mulai: "00:01:00", Selesai: "00:02:00"}, // 60s
				{Mulai: "garbage", Selesai: "00:05:30"},  // skipped
			},
			want: 60,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := TotalDuration(tc.segmen); got != tc.want {
				t.Errorf("TotalDuration() = %v, want %v", got, tc.want)
			}
		})
	}
}
