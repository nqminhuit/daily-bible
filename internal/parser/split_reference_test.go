package parser

import (
	"strings"
	"testing"
)

func TestSeparateGospelReferenceStaysWithinHeaderBlock(t *testing.T) {
	for _, tc := range []struct {
		name, before string
		ok           bool
	}{
		{"adjacent", `<p>Tin Mừng Mc 13,33-37</p>`, true},
		{"summary", `<p>Tin Mừng &nbsp; Mc 13,33-37</p> <p>Summary.</p>`, true},
		{"colon", `<p>Tin Mừng: Mc 13,33-37</p>`, true},
		{"unrelated text", `<p>Tin Mừng references Mc 13,33-37 in a commentary.</p>`, false},
		{"nearer disagreement", `<p>Tin Mừng Mc 13,33-37</p><p>Tin Mừng Mt 13,33-37</p>`, false},
		{"wrong book", `<p>Tin Mừng Mt 13,33-37</p><p>Summary.</p>`, false},
		{"acclamation", `<p>Tung hô Tin Mừng Mc 13,33-37</p><p>Summary.</p>`, false},
		{"first reading", `<p>Bài đọc 1 Mc 13,33-37</p><p>Summary.</p>`, false},
		{"too far", `<p>Tin Mừng Mc 13,33-37</p><p>Other.</p><p>Summary.</p>`, false},
		{"wrapper boundary", `<p>Tin Mừng Mc 13,33-37</p><div>Boundary.</div>`, false},
		{"no reference", `<p>Tin Mừng</p><p>Summary.</p>`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := `<div class="section__content">` + tc.before + `<p>✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Mác-cô.</p><p><sup>33</sup>Test verse.</p></div>`
			section, ref, err := ExtractGospel(input)
			if !tc.ok {
				if err == nil {
					t.Fatal("borrowed invalid reference", ref)
				}
				return
			}
			if err != nil || ref != "Mc 13,33-37" || !strings.Contains(section, "Test verse.") || strings.Contains(section, "Summary.") {
				t.Fatal(section, ref, err)
			}
		})
	}
}
