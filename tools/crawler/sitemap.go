package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ParseSitemap reads a standard sitemap XML and returns the list of <loc> URLs.
// Accepts maxUrls to limit the number of URLs returned (after filtering by prefix); if <= 0, no limit is applied.
func parseSitemap(totalUrls int, r io.Reader, prefix string) ([]string, error) {
	var sitemapUrls struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	dec := xml.NewDecoder(r)
	if err := dec.Decode(&sitemapUrls); err != nil {
		return nil, fmt.Errorf("failed to parse sitemap XML: %w", err)
	}
	maxUrls := len(sitemapUrls.URLs)
	if totalUrls > 0 {
		maxUrls = totalUrls
	}
	out := make([]string, 0, maxUrls)
	for _, item := range sitemapUrls.URLs {
		loc := strings.TrimSpace(item.Loc)
		if loc == "" {
			continue
		}
		if prefix == "" {
			return nil, fmt.Errorf("prefix cannot be empty when filtering URLs")
		}
		// TODO: deduplicate URLs if they appear multiple times in the sitemap
		if strings.HasPrefix(loc, prefix) {
			out = append(out, loc)
		}
		if len(out) >= maxUrls {
			break
		}
	}
	return out, nil
}

// FetchSitemapAndParse fetches the sitemap at sitemapURL and parses the <loc> entries.
// Accepts a custom http.Client for testability; if nil, a default client with timeout is used.
// Accepts maxUrls to limit the number of URLs returned (after filtering by prefix); if <= 0, no limit is applied.
func FetchSitemapAndParse(totalUrls int, sitemapURL, prefix string, client *http.Client) ([]string, error) {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Get(sitemapURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sitemap: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	// Read whole body then parse all URLs, then filter to the Vatican prefix.
	// TODO: can we stream the XML and filter as we go to avoid reading the whole body into memory?
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read sitemap body: %w", err)
	}
	urls, err := parseSitemap(totalUrls, strings.NewReader(string(b)), prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sitemap: %w", err)
	}
	return urls, nil
}
