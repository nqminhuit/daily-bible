package main_test

import (
	"os"
	"strings"
	"testing"

	"github.com/minh/daily-bible/internal/parser"
)

func TestExtractGospelFromFixture(t *testing.T) {
	htmlPath := "../../test-data/22mar2026.html"

	b, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read html fixture: %v", err)
	}
	actualContent, ref, err := parser.ExtractGospel(string(b))
	if err != nil {
		t.Fatalf("ExtractGospel error: %v", err)
	}

	if ref != "Ga 11,1-45" {
		t.Fatalf("unexpected reference: got %q, want %q", ref, "Ga 11,1-45")
	}

	expected, err := os.ReadFile("../../test-data/expected.html")
	if err != nil {
		t.Fatalf("read expected content fixture: %v", err)
	}
	strExpected := strings.TrimSpace(string(expected))
	if strings.Compare(actualContent, strExpected) != 0 {
		t.Fatalf("unexpected extracted content, expected: %s, got: %s", strExpected, actualContent)
	}
}

func TestExtractGospelFromFixture_2(t *testing.T) {
	htmlPath := "../../test-data/13jan2025.html"

	b, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read html fixture: %v", err)
	}
	actualContent, ref, err := parser.ExtractGospel(string(b))
	if err != nil {
		t.Fatalf("ExtractGospel error: %v", err)
	}

	want := "Mc 1,14-20"
	if ref != want {
		t.Fatalf("unexpected reference: got %q, want %q", ref, want)
	}

	expected, err := os.ReadFile("../../test-data/expected_13jan2025.html")
	if err != nil {
		t.Fatalf("read expected content fixture: %v", err)
	}
	strExpected := strings.TrimSpace(string(expected))
	if strings.Compare(actualContent, strExpected) != 0 {
		t.Fatalf("unexpected extracted content, expected: %s, got: %s", strExpected, actualContent)
	}
}

func TestFindReadingStartVatican(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Tin Mừng: something", true},
		{"Lời Chúa something", true},
		{"random text", false},
		{"tin mừng lowercase", true},
	}
	for _, c := range cases {
		r := parser.FindReadingStartVatican(c.in)
		if (r >= 0) != c.want {
			t.Fatalf("FindReadingStartVatican(%q) = %d, want presence %v", c.in, r, c.want)
		}
	}
}

func TestExtractGospel_ReferenceVariants(t *testing.T) {
	tests := []struct {
		name       string
		headerLine string
		wantRef    string
	}{
		{
			name:       "space after comma is normalized",
			headerLine: "✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Gio-an. Ga 4, 5-42",
			wantRef:    "Ga 4,5-42",
		},
		{
			name:       "dot separated ranges are preserved",
			headerLine: "✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Mát-thêu Mt 5, 20-22a.27-28.33-34a.37",
			wantRef:    "Mt 5,20-22a.27-28.33-34a.37",
		},
		{
			name:       "book inferred from evangelist when missing in numeric reference",
			headerLine: "✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Mác-cô. 2,1-12",
			wantRef:    "Mc 2,1-12",
		},
		{
			name:       "book and chapter are separated when source has no space",
			headerLine: "✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Gio-an. Ga10,1-10",
			wantRef:    "Ga 10,1-10",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			htmlInput := `<html><body><section><div class="section__content">` +
				`<p>` + tc.headerLine + `</p>` +
				`<p><sup>1</sup> Câu thử nghiệm.</p>` +
				`</div></section></body></html>`
			_, ref, err := parser.ExtractGospel(htmlInput)
			if err != nil {
				t.Fatalf("ExtractGospel returned error: %v", err)
			}
			if ref != tc.wantRef {
				t.Fatalf("unexpected reference: got %q, want %q", ref, tc.wantRef)
			}
		})
	}
}

func TestIsGospelHeaderText(t *testing.T) {
	if parser.IsGospelHeaderText(`Hễ Tin Mừng được loan báo đến đâu trong khắp thiên hạ`) {
		t.Fatalf("expected body sentence not to be detected as gospel header")
	}
	if !parser.IsGospelHeaderText(`✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Lu-ca. Lc 7, 11-17`) {
		t.Fatalf("expected canonical header line to be detected")
	}
}
