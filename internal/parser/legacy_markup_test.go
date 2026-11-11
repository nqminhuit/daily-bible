package parser

import (
	"strings"
	"testing"
)

const legacyHeader = `<p><b>✠ Tin Mừng Chúa Giê-su Ki-tô theo thánh Lu-ca </b>(Lc 11,14-23)</p>`

// 2021 pages: bare or nested verse markers, and a third section after the Gospel.
func TestLegacyMarkupKeepsTheGospelSection(t *testing.T) {
	page := `<section><h2>Bài đọc</h2><div class="section__content"><p>Bài đọc 1</p></div></section>` +
		`<section><h2>Tin Mừng ngày hôm nay</h2><div class="section__content">` + legacyHeader +
		`<p><span>14</span> Khi ấy.</p><p><span><sup>15</sup></span> Nhưng.</p><p><sup><span>16</span> </sup>Kẻ khác.</p></div></section>` +
		`<section><h2>Lời Đức Thánh Cha</h2><div class="section__content"><p>Suy niệm.</p></div></section>`
	section, ref, err := ExtractGospel(page)
	if err != nil {
		t.Fatal(err)
	}
	if ref != "Lc 11,14-23" {
		t.Fatalf("ref = %q", ref)
	}
	for _, s := range []string{"Khi ấy", "Nhưng", "Kẻ khác"} {
		if !strings.Contains(section, s) {
			t.Errorf("missing %q in %q", s, section)
		}
	}
	if strings.Contains(section, "Suy niệm") {
		t.Errorf("took the reflection section: %q", section)
	}
}

func TestSpanWithoutVerseNumberIsNotAVerseParagraph(t *testing.T) {
	section, _, err := ExtractGospel(`<div class="section__content">` + legacyHeader +
		`<p><span>14</span> Khi ấy.</p><p><span>Ghi chú</span> cuối bài.</p></div>`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(section, "Ghi chú") {
		t.Errorf("unexpected trailing paragraph in %q", section)
	}
}

func TestClosingGospelHeaderIsAccepted(t *testing.T) {
	_, ref, err := ExtractGospel(`<div class="section__content"><p>Kết thúc Tin Mừng Chúa Giê-su Ki-tô theo thánh Mát-thêu (Mt 28,16-20)</p><p><sup>16</sup> Khi ấy.</p></div>`)
	if err != nil || ref != "Mt 28,16-20" {
		t.Fatalf("ref = %q, err = %v", ref, err)
	}
}
