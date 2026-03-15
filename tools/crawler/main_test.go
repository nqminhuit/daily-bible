package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHelperFunctions(t *testing.T) {
	// wrapVerseHTML
	in := `<sup>1</sup> Some text <sup class="num"><b>2</b></sup>`
	out := wrapVerseHTML(in)
	if !strings.Contains(out, "{{1}}") || !strings.Contains(out, "{{2}}") {
		t.Fatalf("wrapVerseHTML failed: %q", out)
	}

	// findReadingStart
	if idx := findReadingStart("prefix Tin mừng: here"); idx == -1 {
		t.Fatalf("findReadingStart missed Tin mừng")
	}
	if idx := findReadingStart("something Lời Chúa: more"); idx == -1 {
		t.Fatalf("findReadingStart missed Lời Chúa")
	}
	if idx := findReadingStart("nothing here"); idx != -1 {
		t.Fatalf("findReadingStart false positive")
	}

	// cutBeforeSuyNiem
	s := "start content<h2>should cut<h2>rest"
	if got := cutBeforeSuyNiem(s); strings.Contains(got, "should cut") {
		t.Fatalf("cutBeforeSuyNiem did not cut: %q", got)
	}

	// stripHtmlTags
	html := `<div>Hello <b>World</b> &amp; friends</div>`
	if got := stripHtmlTags(html); !strings.Contains(got, "Hello") || strings.Contains(got, "<b>") {
		t.Fatalf("stripHtmlTags failed: %q", got)
	}
}

func TestExtractAndCleanFromSample(t *testing.T) {
	path := filepath.Join("..", "..", "test-data", "page1_test.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sample failed: %v", err)
	}
	html := string(b)

	h1 := extractH1(html)
	if h1 == "" {
		t.Fatalf("expected non-empty h1 from sample")
	}

	article := extractArticleDetail(html)
	if article == "" {
		t.Fatalf("expected non-empty article detail")
	}

	// find reading start inside article
	idx := findReadingStart(article)
	if idx == -1 {
		t.Fatalf("expected to find reading start in article")
	}
	content := article[idx:]
	clean := cleanText(content)
	if clean == "" {
		t.Fatalf("cleanText returned empty")
	}
	// cleaned text should not contain raw HTML tags or <sup>
	if strings.Contains(clean, "<") || strings.Contains(clean, "</") {
		t.Fatalf("cleaned text still contains tags: %q", clean[:200])
	}
	// should contain a verse marker like {{1}}
	if !strings.Contains(clean, "{{") {
		t.Fatalf("cleaned text missing verse markers: %q", clean[:200])
	}
}

func TestLoadLinksAndProcessed(t *testing.T) {
	temp := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(temp)

	// create links file
	links := []string{"http://a/1", "http://a/2", ""}
	n := filepath.Join("links.txt")
	_ = os.WriteFile(n, []byte(strings.Join(links, "\n")), 0644)
	got, err := loadLinks(n)
	if err != nil {
		t.Fatalf("loadLinks error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 links, got %d", len(got))
	}

	// processed
	p := filepath.Join("processed.txt")
	_ = os.WriteFile(p, []byte("http://a/2\n"), 0644)
	m := loadProcessed(p)
	if !m["http://a/2"] {
		t.Fatalf("loadProcessed missing entry")
	}
}

func TestWritersAndWorkerIntegration(t *testing.T) {
	// Use test HTML files served by a test server
	page1 := filepath.Join("..", "..", "test-data", "page1_test.html")
	b1, _ := os.ReadFile(page1)
	page2 := filepath.Join("..", "..", "test-data", "page2_test.html")
	b2, _ := os.ReadFile(page2)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.Contains(p, "page1") || strings.HasSuffix(p, "/1") {
			w.Write(b1)
			return
		}
		if strings.Contains(p, "page2") || strings.HasSuffix(p, "/2") {
			w.Write(b2)
			return
		}
		w.Write([]byte("<html></html>"))
	}))
	defer s.Close()

	// chdir to temp to avoid touching repo build files
	temp := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(temp)

	// ensure build dir exists (writers create files inside build)
	if err := os.MkdirAll("build", 0755); err != nil {
		t.Fatalf("mkdir build: %v", err)
	}

	// start writers
	resultsCh := make(chan string, 10)
	doneCh := make(chan string, 10)
	missingCh := make(chan string, 10)

	go resultsWriter(resultsCh)
	go processedWriter(doneCh)
	go missingVerseNumWriter(missingCh)

	// prepare http client
	client := &http.Client{Timeout: 2 * time.Second}

	// run worker for two fake URLs
	jobs := make(chan string, 2)
	jobs <- s.URL + "/page1"
	jobs <- s.URL + "/page2"
	close(jobs)

	// set workerSleep to zero to speed up test
	workerSleep = 0

	var wg sync.WaitGroup
	wg.Add(1)
	go worker(client, jobs, resultsCh, doneCh, missingCh, &wg, 2)

	// wait for worker
	wg.Wait()

	// close writer channels so files flush
	close(resultsCh)
	close(missingCh)
	close(doneCh)

	// give writers a moment
	time.Sleep(10 * time.Millisecond)

	// check files exist
	if _, err := os.Stat("build/gospels.txt"); err != nil {
		t.Fatalf("results file missing: %v", err)
	}
	if _, err := os.Stat("build/missing_verse_number.txt"); err != nil {
		t.Fatalf("missing verse file missing: %v", err)
	}
	if _, err := os.Stat("build/processed.txt"); err != nil {
		t.Fatalf("processed file missing: %v", err)
	}

	// basic contents check
	b, _ := os.ReadFile("build/processed.txt")
	if len(strings.TrimSpace(string(b))) == 0 {
		t.Fatalf("processed file seems empty")
	}
}

func TestExtractEdgeCasesAndWritersFlush(t *testing.T) {
	// extractH1 edge cases
	if v := extractH1(""); v != "" {
		t.Fatalf("expected empty for empty html, got %q", v)
	}
	if v := extractH1("<h1>No close"); v != "" {
		t.Fatalf("expected empty for unclosed h1, got %q", v)
	}
	if v := extractH1("<h1>OK</h1> rest"); v != "OK" {
		t.Fatalf("expected OK, got %q", v)
	}

	// extractArticleDetail with nested divs
	html := `<div class="article-detail"><div><p>inner</p></div></div>`
	if got := extractArticleDetail(html); !strings.Contains(got, "inner") {
		t.Fatalf("extractArticleDetail failed: %q", got)
	}

	// test resultsWriter flush behavior (flush every 10 writes)
	temp := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(temp)
	_ = os.MkdirAll("build", 0755)

	resCh := make(chan string, 20)
	go resultsWriter(resCh)
	for i := range 15 {
		resCh <- fmt.Sprintf("LINE%02d", i)
	}
	close(resCh)
	// wait for flush
	time.Sleep(10 * time.Millisecond)
	b, err := os.ReadFile("build/gospels.txt")
	if err != nil {
		t.Fatalf("read results file: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("expected non-empty results file")
	}
}

func TestWorkerSkipsNon200(t *testing.T) {
	// server that returns 500
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("err"))
	}))
	defer s.Close()

	// chdir to temp
	temp := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(temp)
	_ = os.MkdirAll("build", 0755)

	resultsCh := make(chan string, 10)
	doneCh := make(chan string, 10)
	missingCh := make(chan string, 10)
	go resultsWriter(resultsCh)
	go processedWriter(doneCh)
	go missingVerseNumWriter(missingCh)

	client := &http.Client{Timeout: 2 * time.Second}
	jobs := make(chan string, 1)
	jobs <- s.URL + "/bad"
	close(jobs)

	workerSleep = 0
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(client, jobs, resultsCh, doneCh, missingCh, &wg, 1)
	wg.Wait()

	close(resultsCh)
	close(doneCh)
	close(missingCh)
	// wait
	time.Sleep(10 * time.Millisecond)
	// processed file should exist but be empty
	b, _ := os.ReadFile("build/processed.txt")
	if len(strings.TrimSpace(string(b))) != 0 {
		t.Fatalf("expected processed file to be empty when worker skipped non-200; got %q", string(b))
	}
}
