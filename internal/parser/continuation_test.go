package parser

import (
	"strings"
	"testing"
)

const continuationHeader = `<p>✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Lu-ca. Lc 7,31-35</p>`

func TestUnnumberedParagraphContinuesOnlyBeforeANumberedOne(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want, deny []string
	}{
		{"poetry between verses", `<p><sup>32</sup> Họ nói :</p> <p>‘Tụi tôi thổi sáo,<br/> mà các anh không nhảy múa.’</p> <p><sup>33</sup> Thật vậy.</p>`,
			[]string{"Tụi tôi thổi sáo", "Thật vậy"}, nil},
		{"trailing unnumbered text is not taken", `<p><sup>32</sup> Họ nói :</p> <p>Ghi chú cuối bài.</p>`,
			[]string{"Họ nói"}, []string{"Ghi chú"}},
		{"two unnumbered paragraphs stop the section", `<p><sup>32</sup> Họ nói :</p> <p>Một.</p> <p>Hai.</p> <p><sup>33</sup> Thật vậy.</p>`,
			[]string{"Họ nói"}, []string{"Một", "Thật vậy"}},
		{"blank paragraph ends the section", `<p><sup>32</sup> Họ nói :</p> <p> </p> <p><sup>33</sup> Thật vậy.</p>`,
			[]string{"Họ nói"}, []string{"Thật vậy"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			section, _, err := ExtractGospel(`<div class="section__content">` + continuationHeader + tc.body + `</div>`)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range tc.want {
				if !strings.Contains(section, s) {
					t.Errorf("missing %q in %q", s, section)
				}
			}
			for _, s := range tc.deny {
				if strings.Contains(section, s) {
					t.Errorf("unexpected %q in %q", s, section)
				}
			}
		})
	}
}
