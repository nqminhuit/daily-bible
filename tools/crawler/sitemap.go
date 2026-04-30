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
	if prefix == "" {
		return nil, fmt.Errorf("prefix cannot be empty when filtering URLs")
	}

	dec := xml.NewDecoder(r)
	maxUrls := totalUrls
	if maxUrls <= 0 {
		maxUrls = -1
	}

	out := make([]string, 0)
	seen := make(map[string]struct{})
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse sitemap XML: %w", err)
		}

		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "loc" {
			continue
		}

		var loc string
		if err := dec.DecodeElement(&loc, &start); err != nil {
			return nil, fmt.Errorf("failed to decode sitemap loc: %w", err)
		}
		loc = strings.TrimSpace(loc)
		if loc == "" || !strings.HasPrefix(loc, prefix) {
			continue
		}
		if _, exists := seen[loc]; exists {
			continue
		}
		seen[loc] = struct{}{}
		out = append(out, loc)
		if maxUrls > 0 && len(out) >= maxUrls {
			return out, nil
		}
	}
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
	urls, err := parseSitemap(totalUrls, resp.Body, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sitemap: %w", err)
	}
	return urls, nil
}
