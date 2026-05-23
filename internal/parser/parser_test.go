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
