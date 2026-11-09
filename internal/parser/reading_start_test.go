package parser

import "testing"

func TestFindReadingStartVatican(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int
	}{
		{"Tin Mừng", 0},
		{"Tin Mừng:", 0},
		{"Lời Chúa", 0},
		{"Lời Chúa:", 0},
		{"prefix TIN MỪNG: reading", 7},
		{"prefix LỜI CHÚA: reading", 7},
		{"Lời Chúa: Tin Mừng:", len("Lời Chúa: ")}, // Gospel marker retains priority.
		{"no reading marker", -1},
		{"", -1},
	} {
		t.Run(tc.input, func(t *testing.T) {
			if got := FindReadingStartVatican(tc.input); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
