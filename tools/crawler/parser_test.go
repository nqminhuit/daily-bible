package main

import (
	"os"
	"strings"
	"testing"
)

func TestExtractGospelFromFixture(t *testing.T) {
	htmlPath := "../../test-data/22mar2026.html"

	b, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read html fixture: %v", err)
	}
	actualContent, ref, err := ExtractGospel(string(b))
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
		t.Fatalf("unexpected empty extracted content, expected: %s, got: %s", strExpected, actualContent)
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
		r := findReadingStartVatican(c.in)
		if (r >= 0) != c.want {
			t.Fatalf("findReadingStartVatican(%q) = %d, want presence %v", c.in, r, c.want)
		}
	}
}
