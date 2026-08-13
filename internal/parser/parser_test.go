package parser

import "testing"

func TestCanonicalizeReference(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{
			name: "spaces around comma removed",
			ref:  "Ga 4 , 5-42",
			want: "Ga 4,5-42",
		},
		{
			name: "spaces around dash removed",
			ref:  "Mt 5, 20 - 22a",
			want: "Mt 5,20-22a",
		},
		{
			name: "spaces around dots removed",
			ref:  "Mt 5, 20 . 27",
			want: "Mt 5,20.27",
		},
		{
			name: "book and chapter without space inserted",
			ref:  "Ga10,1-10",
			want: "Ga 10,1-10",
		},
		{
			name: "1 Cor without space",
			ref:  "1Cor 1,1-2",
			want: "1Cor 1,1-2",
		},
		{
			name: "non-breaking space normalized",
			ref:  "Mc\u00A01,14-20",
			want: "Mc 1,14-20",
		},
		{
			name: "already canonical",
			ref:  "Mt 5,20-22a.27-28",
			want: "Mt 5,20-22a.27-28",
		},
		{
			name: "en-dash normalized to hyphen",
			ref:  "Mt 18,21\u201319,1",
			want: "Mt 18,21-19,1",
		},
		{
			name: "em-dash normalized to hyphen",
			ref:  "Lc 1,1\u20142,5",
			want: "Lc 1,1-2,5",
		},
		{
			name: "cross-chapter range",
			ref:  "Mt 18,21-19,1",
			want: "Mt 18,21-19,1",
		},
		{
			name: "empty string",
			ref:  "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalizeReference(tt.ref)
			if got != tt.want {
				t.Fatalf("canonicalizeReference(%q) = %q, want %q", tt.ref, got, tt.want)
			}
		})
	}
}

func TestBibleRefReMatchesCrossChapterRange(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"✠ Tin Mừng Chúa Giê-su Ki-tô theo thánh Mát-thêu. Mt 18,21-19,1", "Mt 18,21-19,1"},
		{"✠ Tin Mừng theo thánh Lu-ca. Lc 1,39-56", "Lc 1,39-56"},
		{"✠ Tin Mừng theo thánh Mát-thêu. Mt 5,20-22a.27-28", "Mt 5,20-22a.27-28"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			normalized := NormalizeSpaces(tt.input)
			got := bibleRefRe.FindString(normalized)
			canonical := canonicalizeReference(got)
			if canonical != tt.want {
				t.Fatalf("from %q got %q, want %q", tt.input, canonical, tt.want)
			}
		})
	}
}
