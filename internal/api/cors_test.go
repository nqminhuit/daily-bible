package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			called := false
			h := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("response"))
			}))
			req := httptest.NewRequest(method, "/api/v1/random", nil)
			req.Header.Set("Origin", "https://example.test")
			if method == http.MethodOptions {
				req.Header.Set("Access-Control-Request-Method", http.MethodGet)
				req.Header.Set("Access-Control-Request-Headers", "Content-Type")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
				t.Fatalf("allow origin = %q", got)
			}
			if method == http.MethodOptions {
				if called || w.Code != http.StatusNoContent || w.Body.Len() != 0 {
					t.Fatalf("preflight invoked handler or produced unexpected response: called=%v status=%d body=%q", called, w.Code, w.Body.String())
				}
				if w.Header().Get("Access-Control-Allow-Methods") != "GET, OPTIONS" || w.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
					t.Fatalf("missing preflight headers: %v", w.Header())
				}
			} else {
				if !called || w.Code != http.StatusOK || w.Body.String() != "response" {
					t.Fatalf("normal response changed: called=%v status=%d body=%q", called, w.Code, w.Body.String())
				}
				if w.Header().Get("Access-Control-Allow-Methods") != "" || w.Header().Get("Access-Control-Allow-Headers") != "" {
					t.Fatalf("preflight-only headers on normal response: %v", w.Header())
				}
			}
		})
	}
}
