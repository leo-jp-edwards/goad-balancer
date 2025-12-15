package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	ts := httptest.NewServer(newHandler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: go %d want %d", resp.StatusCode, http.StatusOK)
	}

	if ct := resp.Header.Get("Content-Type"); ct == "" {
		t.Fatalf("missing Content-Type header")
	}

	var body struct {
		Status string `json:"status"`
		Time   string `json:"time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("time field is not RFC3339Nano: %q (%v)", body.Time, err)
	}
}

func TestHostRouting(t *testing.T) {
	ts := httptest.NewServer(newHandler())
	t.Cleanup(ts.Close)

	client := ts.Client()

	cases := []struct {
		description    string
		host           string
		expectedStatus int
		expectedRoute  string
	}{
		{
			description:    "mango server",
			host:           "mango.com",
			expectedStatus: 200,
			expectedRoute:  "site-mango",
		},
		{
			description:    "apple server",
			host:           "apple.com",
			expectedStatus: 200,
			expectedRoute:  "site-apple",
		},
		{
			description:    "missing server",
			host:           "notmango.com",
			expectedStatus: 404,
			expectedRoute:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}

			req.Host = tc.host

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			t.Cleanup(func() { _ = resp.Body.Close() })

			if resp.StatusCode != tc.expectedStatus {
				t.Fatalf("unexpected status for host %q: got %d want %d", tc.host, resp.StatusCode, tc.expectedStatus)
			}

			if tc.expectedStatus != http.StatusOK {
				return
			}

			var body struct {
				Host  string `json:"host"`
				Route string `json:"route"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode response JSON: %v", err)
			}

			if body.Route != tc.expectedRoute {
				t.Fatalf("unexpected route: got %q want %q", body.Route, tc.expectedRoute)
			}

			if body.Host == "" {
				t.Fatalf("expected ost in response, got empty")
			}
		})
	}
}
