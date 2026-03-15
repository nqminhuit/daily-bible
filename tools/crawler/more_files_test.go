package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Tests that use additional test-data files: no_verse_markers* and use_LoiChua_instead*

func TestNoVerseMarkersFiles(t *testing.T) {
	b1p := filepath.Join("..", "..", "test-data", "no_verse_markers_test.html")
	b2p := filepath.Join("..", "..", "test-data", "no_verse_markers2_test.html")
	b1, err := os.ReadFile(b1p)
	if err != nil {
		t.Fatalf("read b1: %v", err)
	}
	b2, err := os.ReadFile(b2p)
	if err != nil {
		t.Fatalf("read b2: %v", err)
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.Contains(p, "no1") || strings.HasSuffix(p, "/1") {
			w.Write(b1)
			return
		}
		if strings.Contains(p, "no2") || strings.HasSuffix(p, "/2") {
			w.Write(b2)
			return
		}
		w.Write([]byte("<html></html>"))
	}))
	defer s.Close()

	// run in temp dir
	temp := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(temp)
	_ = os.MkdirAll("build", 0755)

	// reset counters
	atomic.StoreInt64(&checked, 0)
	atomic.StoreInt64(&matched, 0)
	atomic.StoreInt64(&missingVerse, 0)

	// start writers
	resultsCh := make(chan string, 10)
	doneCh := make(chan string, 10)
	missingCh := make(chan string, 10)
	go resultsWriter(resultsCh)
	go processedWriter(doneCh)
	go missingVerseNumWriter(missingCh)

	client := &http.Client{Timeout: 2 * time.Second}
	jobs := make(chan string, 2)
	jobs <- s.URL + "/no1"
	jobs <- s.URL + "/no2"
	close(jobs)

	workerSleep = 0
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(client, jobs, resultsCh, doneCh, missingCh, &wg, 2)
	wg.Wait()

	close(resultsCh)
	close(missingCh)
	close(doneCh)
	// allow writers to flush
	time.Sleep(20 * time.Millisecond)

	// missing file should exist and contain entries
	mb, err := os.ReadFile("build/missing_verse_number.txt")
	if err != nil {
		t.Fatalf("missing verse file not created: %v", err)
	}
	if len(strings.TrimSpace(string(mb))) == 0 {
		t.Fatalf("expected missing verse file to contain entries")
	}

	// results file should either be absent or empty (no matched verses)
	rb, _ := os.ReadFile("build/gospels.txt")
	if len(strings.TrimSpace(string(rb))) != 0 {
		t.Fatalf("expected results file to be empty when no verse markers, got %q", string(rb)[:200])
	}
}

func TestUseLoiChuaFiles(t *testing.T) {
	b1p := filepath.Join("..", "..", "test-data", "use_LoiChua_instead_TinMung_test.html")
	b2p := filepath.Join("..", "..", "test-data", "use_LoiChua_instead_TinMung2_test.html")
	b1, err := os.ReadFile(b1p)
	if err != nil {
		t.Fatalf("read b1: %v", err)
	}
	b2, err := os.ReadFile(b2p)
	if err != nil {
		t.Fatalf("read b2: %v", err)
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.Contains(p, "lc1") || strings.HasSuffix(p, "/1") {
			w.Write(b1)
			return
		}
		if strings.Contains(p, "lc2") || strings.HasSuffix(p, "/2") {
			w.Write(b2)
			return
		}
		w.Write([]byte("<html></html>"))
	}))
	defer s.Close()

	// run in temp dir
	temp := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(temp)
	_ = os.MkdirAll("build", 0755)

	// reset counters
	atomic.StoreInt64(&checked, 0)
	atomic.StoreInt64(&matched, 0)
	atomic.StoreInt64(&missingVerse, 0)

	// start writers
	resultsCh := make(chan string, 10)
	doneCh := make(chan string, 10)
	missingCh := make(chan string, 10)
	go resultsWriter(resultsCh)
	go processedWriter(doneCh)
	go missingVerseNumWriter(missingCh)

	client := &http.Client{Timeout: 2 * time.Second}
	jobs := make(chan string, 2)
	jobs <- s.URL + "/lc1"
	jobs <- s.URL + "/lc2"
	close(jobs)

	workerSleep = 0
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(client, jobs, resultsCh, doneCh, missingCh, &wg, 2)
	wg.Wait()

	close(resultsCh)
	close(missingCh)
	close(doneCh)
	// allow writers to flush
	time.Sleep(20 * time.Millisecond)

	// results file should exist and contain matched entries
	rb, err := os.ReadFile("build/gospels.txt")
	if err != nil {
		t.Fatalf("results file not created: %v", err)
	}
	if !strings.Contains(string(rb), "TITLE:") || !strings.Contains(string(rb), "URL:") {
		t.Fatalf("results file missing expected markers, got: %q", string(rb)[:200])
	}

	// processed file should list the urls
	pb, err := os.ReadFile("build/processed.txt")
	if err != nil {
		t.Fatalf("processed file missing: %v", err)
	}
	if len(strings.TrimSpace(string(pb))) == 0 {
		t.Fatalf("processed file empty")
	}
}
