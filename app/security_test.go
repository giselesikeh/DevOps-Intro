package main

import (
	"net/http"
	"testing"
)

func TestSecurityHeaders_AppliedToAllRoutes(t *testing.T) {
	srv := newTestServer(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "existing route", method: http.MethodGet, path: "/health"},
		{name: "not found route", method: http.MethodGet, path: "/does-not-exist"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, srv, tc.method, tc.path, nil)

			wantHeaders := map[string]string{
				"Cache-Control":           "no-store",
				"Pragma":                  "no-cache",
				"Expires":                 "0",
				"X-Content-Type-Options":  "nosniff",
				"X-Frame-Options":         "DENY",
				"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
				"Referrer-Policy":         "no-referrer",
			}

			for name, want := range wantHeaders {
				if got := rec.Header().Get(name); got != want {
					t.Fatalf("%s header = %q, want %q", name, got, want)
				}
			}
		})
	}
}
