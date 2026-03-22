package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cst "github.com/minh/daily-bible/internal/constants"
)

const sampleSitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://www.vaticannews.va/vi/loi-chua-hang-ngay/2026/03/22.html</loc>
  </url>
  <url>
    <loc>https://www.vaticannews.va/vi/some-other-section/2026/03/22.html</loc>
  </url>
  <url>
    <loc>https://example.com/other</loc>
  </url>
</urlset>`

func TestParseSitemap(t *testing.T) {
	urls, err := parseSitemap(0, strings.NewReader(sampleSitemap), cst.VaticanPrefix)
	if err != nil {
		t.Fatalf("ParseSitemap failed: %v", err)
	}
	if len(urls) != 1 {
		t.Fatalf("expected 1 urls, got %d", len(urls))
	}
	if urls[0] != "https://www.vaticannews.va/vi/loi-chua-hang_ngay/2026/03/22.html" && urls[0] != "https://www.vaticannews.va/vi/loi-chua-hang-ngay/2026/03/22.html" {
		t.Fatalf("unexpected first url: %s", urls[0])
	}
}

func TestFilterURLs(t *testing.T) {
	filtered, err := parseSitemap(0, strings.NewReader(sampleSitemap), cst.VaticanPrefix)
	if err != nil {
		t.Fatalf("parseSitemap failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered url, got %d", len(filtered))
	}
	if !strings.HasPrefix(filtered[0], cst.VaticanPrefix) {
		t.Fatalf("filtered url does not have prefix: %s", filtered[0])
	}
}

func TestFetchSitemapAndParse(t *testing.T) {
	h := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleSitemap))
	}
	srv := httptest.NewServer(http.HandlerFunc(h))
	defer srv.Close()

	client := srv.Client()
	urls, err := FetchSitemapAndParse(0, srv.URL+"/sitemap.xml", cst.VaticanPrefix, client)
	if err != nil {
		t.Fatalf("FetchSitemapAndParse failed: %v", err)
	}
	if len(urls) != 1 {
		t.Fatalf("expected 1 urls from server, got %d", len(urls))
	}
}
