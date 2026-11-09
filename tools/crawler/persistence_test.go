package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestCrawlSink(t *testing.T, dir string) *crawlSink {
	t.Helper()
	sink, err := openCrawlSink(filepath.Join(dir, "gospels.txt"), filepath.Join(dir, "processed.txt"), filepath.Join(dir, "failed.txt"), filepath.Join(dir, "missing_verse_number.txt"), func() {})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sink.close(); err != nil {
			t.Error(err)
		}
	})
	return sink
}

// Collect worker output independently of persistence for parser/HTTP tests.
func collectTestResult(results, done, missing, failed chan<- string) func(crawlResult) error {
	return func(r crawlResult) error {
		if r.failed {
			failed <- r.url
			return nil
		}
		if r.missing {
			missing <- r.payload
		} else {
			results <- r.payload
		}
		done <- r.url
		return nil
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

func TestCrawlSinkPayloadFailure(t *testing.T) {
	for _, missing := range []bool{false, true} {
		for _, size := range []int{10, 8192} { // buffered flush failure and direct write failure
			name := "results"
			if missing {
				name = "missing"
			}
			t.Run(fmt.Sprintf("%s/bytes=%d", name, size), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				failure := errors.New("disk full")
				var processed, failed bytes.Buffer
				bad := bufio.NewWriter(writerFunc(func([]byte) (int, error) { return 0, failure }))
				sink := &crawlSink{results: bad, missing: bad, processed: bufio.NewWriter(&processed), failed: bufio.NewWriter(&failed), cancel: cancel}
				err := sink.persist(crawlResult{url: "retry-me", payload: strings.Repeat("x", size), missing: missing})
				if !errors.Is(err, failure) {
					t.Fatalf("expected disk failure, got %v", err)
				}
				if ctx.Err() == nil {
					t.Fatal("persistence failure did not cancel workers")
				}
				if err2 := sink.persist(crawlResult{url: "also-retry", payload: "next"}); err2 != err {
					t.Fatalf("expected sticky failure, got %v", err2)
				}
				if processed.Len() != 0 || failed.Len() != 0 {
					t.Fatal("unwritten payload was added to a skip-state file")
				}
			})
		}
	}
}

func TestCrawlSinkPayloadBeforeProcessed(t *testing.T) {
	for _, missing := range []bool{false, true} {
		var payload bytes.Buffer
		state := writerFunc(func(p []byte) (int, error) {
			if payload.String() != "payload" {
				t.Fatalf("processed entry written before payload flush: %q", payload.String())
			}
			if string(p) != "url\n" {
				t.Fatalf("unexpected processed entry %q", p)
			}
			return len(p), nil
		})
		w := bufio.NewWriter(&payload)
		sink := &crawlSink{results: w, missing: w, processed: bufio.NewWriter(state), cancel: func() { t.Error("unexpected cancellation") }}
		if err := sink.persist(crawlResult{url: "url", payload: "payload", missing: missing}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCrawlSinkStateFailure(t *testing.T) {
	failure := errors.New("state write failure")
	for _, failed := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		bad := bufio.NewWriter(writerFunc(func([]byte) (int, error) { return 0, failure }))
		sink := &crawlSink{results: bufio.NewWriter(io.Discard), processed: bad, failed: bad, cancel: cancel}
		if err := sink.persist(crawlResult{url: "url", payload: "payload", failed: failed}); !errors.Is(err, failure) {
			t.Fatalf("got %v", err)
		}
		if ctx.Err() == nil {
			t.Error("expected cancellation")
		}
		cancel()
	}
}

func TestCrawlSinkCloseErrors(t *testing.T) {
	failure := errors.New("close failed")
	closed := 0
	sink := &crawlSink{files: []io.Closer{
		closerFunc(func() error { closed++; return failure }),
		closerFunc(func() error { closed++; return nil }),
	}}
	if err := sink.close(); !errors.Is(err, failure) {
		t.Fatalf("got %v", err)
	}
	if closed != 2 {
		t.Fatalf("closed %d files, want 2", closed)
	}
}

func TestRunCrawlerOutputOpenFailure(t *testing.T) {
	dir := t.TempDir()
	// Opening a directory as an output must return an error, not panic or fetch.
	err := runCrawler(0, filepath.Join(dir, "gospels.txt"), dir, filepath.Join(dir, "failed.txt"), filepath.Join(dir, "missing.txt"), "invalid sitemap", "prefix", 0)
	if err == nil || !strings.Contains(err.Error(), "open crawler output") {
		t.Fatalf("got %v", err)
	}
}

func TestWorkerCancellation(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobs := make(chan string, 1)
	jobs <- srv.URL
	close(jobs)
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(workerConfig{ctx: ctx, client: srv.Client(), jobs: jobs, wg: &wg, counters: &crawlCounters{}, persist: func(crawlResult) error { t.Error("cancelled request should not persist"); return nil }})
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	if err := waitWithTimeout(&wg, 2*time.Second); err != nil {
		t.Fatal(err)
	}
}
