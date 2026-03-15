package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBibleLinks_UsesLocalHTMLAndWritesLinks(t *testing.T) {
	// load the sample HTML from repo test-data
	htmlPath := filepath.Join("..", "..", "test-data", "biblelinks_test.html")
	b, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("failed to read test html: %v", err)
	}

	// start a test server that serves the HTML for page=1 and empty for others
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("page")
		if q == "1" || q == "" {
			w.Write(b)
			return
		}
		// return an HTML page without links
		w.Write([]byte("<html></html>"))
	}))
	defer s.Close()

	// override baseURL and sleepDur so test is fast and uses test server
	baseURL = s.URL
	sleepDur = 0

	// run in a temp dir so we don't touch repo build/ files
	temp := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}

	// ensure build dir exists
	if err := os.MkdirAll("build", 0755); err != nil {
		t.Fatal(err)
	}
	// run main (should exit after page 2 finds no links)
	main()

	// read output file
	out := filepath.Join("build", "bible-links.txt")
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		t.Fatalf("expected some links, got none")
	}
	// ensure links are absolute and point to our test server host
	for _, l := range lines {
		if !strings.HasPrefix(l, baseURL+"/bai-viet/") {
			t.Fatalf("unexpected link format: %q", l)
		}
	}
}
