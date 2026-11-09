package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRefreshDifferenceAndOutageDoNotInvalidateWarmCache(t *testing.T) {
	for _, outage := range []bool{false, true} {
		t.Run(fmt.Sprint("outage=", outage), func(t *testing.T) {
			client, run := cacheFixture(t, true)
			client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if outage {
					return nil, fmt.Errorf("simulated network outage")
				}
				year := "2025"
				if strings.Contains(r.URL.Path, "/2027/") {
					year = "2027"
				}
				page := fmt.Sprintf(`<html><head><title>Lời Chúa ngày 9 tháng 11 %s</title></head><body><section><div class="section__content"><p>✠Bài trích Tin Mừng Chúa Giê-su Ki-tô theo thánh Lu-ca. Lc 9,23-26</p><p><sup>23</sup>Text</p></div></section></body></html>`, year)
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(page))}, nil
			})
			stats, err := run(false, true)
			requests := 2
			if outage {
				requests = 6
			} // three bounded attempts per date
			if err == nil || stats.HTTPRequests != requests || stats.CacheHits != 2 || stats.Unresolved != 0 {
				t.Fatalf("refresh: %+v %v", stats, err)
			}
			if !outage && stats.Differences != 2 {
				t.Fatalf("differences not reported: %+v", stats)
			}
			stats, err = run(false, false)
			if err != nil || stats.HTTPRequests != 0 || stats.CacheHits != 2 || stats.Unresolved != 0 {
				t.Fatalf("cached availability poisoned: %+v %v", stats, err)
			}
		})
	}
}
