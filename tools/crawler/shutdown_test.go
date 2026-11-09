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
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunCrawlerPersistenceFailureStopsWorkers(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing=%v", missing), func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.Mkdir("build", 0755); err != nil {
				t.Fatal(err)
			}
			page := `<section><div class="section__content"><p>✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Mác-cô. Mc 1,14-20</p><p><sup>14</sup>Verse one</p></div></section>`
			if missing {
				page = strings.ReplaceAll(page, "<sup>14</sup>", "")
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/sitemap.xml" {
					fmt.Fprint(w, "<urlset>")
					for i := range 1000 {
						fmt.Fprintf(w, "<url><loc>http://%s/page%d</loc></url>", r.Host, i)
					}
					fmt.Fprint(w, "</urlset>")
					return
				}
				fmt.Fprint(w, page)
			}))
			defer srv.Close()

			failure := errors.New("injected flush failure")
			var processed, failed bytes.Buffer
			closed := 0
			opener := func(_, _, _, _ string, cancel context.CancelFunc) (*crawlSink, error) {
				bad := bufio.NewWriter(writerFunc(func([]byte) (int, error) { return 0, failure }))
				return &crawlSink{
					results: bad, missing: bad,
					processed: bufio.NewWriter(&processed), failed: bufio.NewWriter(&failed), cancel: cancel,
					files: []io.Closer{closerFunc(func() error { closed++; return nil })},
				}, nil
			}
			done := make(chan error, 1)
			go func() {
				done <- runCrawlerWithSinkOpener(0, "out", "processed", "failed", "missing", srv.URL+"/sitemap.xml", srv.URL+"/", time.Hour, opener)
			}()
			select {
			case err := <-done:
				if !errors.Is(err, failure) {
					t.Fatalf("expected persistence error, got %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("crawler did not stop after persistence failure")
			}
			if processed.Len() != 0 || failed.Len() != 0 {
				t.Fatal("failed payloads must remain retryable")
			}
			if closed != 1 {
				t.Fatalf("closed %d outputs, want 1", closed)
			}
		})
	}
}

func TestRunCrawlerSetupFailureClosesOutputs(t *testing.T) {
	for _, sitemapFailure := range []bool{false, true} {
		t.Run(fmt.Sprintf("sitemapFailure=%v", sitemapFailure), func(t *testing.T) {
			t.Chdir(t.TempDir()) // No build directory: writing links fails if sitemap succeeds.
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if sitemapFailure {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				fmt.Fprintf(w, "<urlset><url><loc>http://%s/page</loc></url></urlset>", r.Host)
			}))
			defer srv.Close()
			closeFailure := errors.New("injected close failure")
			closed := false
			opener := func(_, _, _, _ string, cancel context.CancelFunc) (*crawlSink, error) {
				return &crawlSink{cancel: cancel, files: []io.Closer{closerFunc(func() error { closed = true; return closeFailure })}}, nil
			}
			err := runCrawlerWithSinkOpener(0, "out", "processed", "failed", "missing", srv.URL+"/sitemap.xml", srv.URL+"/", 0, opener)
			if !closed || !errors.Is(err, closeFailure) {
				t.Fatalf("output not closed or error lost: %v", err)
			}
			want := "write links file"
			if sitemapFailure {
				want = "fetch and parse sitemap"
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("setup error lost: %v", err)
			}
		})
	}
}
