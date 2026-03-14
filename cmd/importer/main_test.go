package main

import "testing"

type exp struct {
	book           string
	cs, vs, ce, ve int
}

func TestParseReference(t *testing.T) {
	cases := map[string]exp{
		"Mt 26,14-27,66": {"Mt", 26, 14, 27, 66},
		"Ga 10,31-42":    {"Ga", 10, 31, 10, 42},
		"Mt 1,1-2":       {"Mt", 1, 1, 1, 2},
		"Jn 3,16":        {"Jn", 3, 16, 3, 16},
		"Mt 5":           {"Mt", 5, 0, 5, 0},
		"":               {"", 0, 0, 0, 0},
	}

	for ref, e := range cases {
		t.Run(ref, func(t *testing.T) {
			b, cs, vs, ce, ve := parseReference(ref)
			if b != e.book || cs != e.cs || vs != e.vs || ce != e.ce || ve != e.ve {
				t.Fatalf("parseReference(%q) => %q,%d,%d,%d,%d; want %q,%d,%d,%d,%d",
					ref, b, cs, vs, ce, ve, e.book, e.cs, e.vs, e.ce, e.ve)
			}
		})
	}
}
