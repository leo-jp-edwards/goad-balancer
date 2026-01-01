package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func main() {
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           newHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}

func newHandler() http.Handler {
	hostRoutes := map[string]string{
		"mango.com": "site-mango",
		"apple.com": "site-apple",
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(healthResponse{
			Status: "ok",
			Time:   time.Now().Format(time.RFC3339Nano),
		})
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		host := canonicalHost(r.Host)

		route, ok := hostRoutes[host]
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(hostResponse{
			Host:  host,
			Route: route,
		})
	})

	return mux
}

// canonicalHost strips an optional :port and normalises case
// it also tolerates bracketed IPv6 hosts like "[::1]:8080"
func canonicalHost(h string) string {
	h = strings.TrimSpace(strings.ToLower(h))
	if h == "" {
		return ""
	}
	// if it looks like host:port or [v6]:port, split
	if host, _, err := net.SplitHostPort(h); err == nil {
		// net.SplitHostPort keeps brackets off the returned host for IPv6
		return strings.Trim(host, "[]")
	}
	// otherwise it may already be a host
	return strings.Trim(h, "[]")
}
