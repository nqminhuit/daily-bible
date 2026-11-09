package parser

import "testing"

func TestPageDateMustComeFromValidTitle(t *testing.T) {
	for _, tc := range []struct{ html, want string }{
		{`<title>Tin Mừng và Lời Chúa ngày 9 tháng 11 2026 - Vatican News</title>`, "2026-11-09"},
		{`<meta content="ngày 9 tháng 11 2026"><title>Tin Mừng ngày 20 tháng 12 2025</title>`, "2025-12-20"},
		{`<title>Lời Chúa ngày 31 tháng 2 2026</title>`, ""},
		{`<title>Lời Chúa ngày 29 tháng 2 2024</title>`, "2024-02-29"},
		{`<title>Lời Chúa ngày 29 tháng 2 2025</title>`, ""},
		{`<title>Undated page</title><p>ngày 9 tháng 11 2026</p>`, ""},
		{`<p>ngày 9 tháng 11 2026</p>`, ""},
	} {
		got, err := ExtractPageDate(tc.html)
		if tc.want == "" {
			if err == nil {
				t.Fatalf("accepted invalid page: %s", tc.html)
			}
		} else if err != nil || got != tc.want {
			t.Fatalf("got %q, %v; want %q", got, err, tc.want)
		}
	}
}
