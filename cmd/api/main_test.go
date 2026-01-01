package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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

func TestRoutingAgainstMultipleBackends(t *testing.T) {
	ctx := context.Background()

	// define the services we want to spin up
	services := []struct {
		name     string
		expected string
	}{
		{"mango", "This is the mango application"},
		{"apple", "This is the apple application"},
		{"banana", "This is the banana application"},
	}

	// build the image once from the app directory
	appDir, err := filepath.Abs("../../app")
	if err != nil {
		t.Fatalf("failed to get app directory: %v", err)
	}

	var containers []testcontainers.Container
	var urls []string

	for _, svc := range services {
		req := testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    appDir,
				Dockerfile: "Dockerfile",
			},
			ExposedPorts: []string{"5000/tcp"},
			Env: map[string]string{
				"APP": svc.name,
			},
			WaitingFor: wait.ForHTTP("/").WithPort("5000/tcp").WithStartupTimeout(30 * time.Second),
		}

		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			t.Fatalf("failed to start %s container: %v", svc.name, err)
		}

		containers = append(containers, container)

		mappedPort, err := container.MappedPort(ctx, "5000")
		if err != nil {
			t.Fatalf("failed to get mapped port for %s: %v", svc.name, err)
		}

		host, err := container.Host(ctx)
		if err != nil {
			t.Fatalf("failed to get host for %s: %v", svc.name, err)
		}

		url := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())
		urls = append(urls, url)

		t.Logf("%s server running at: %s", svc.name, url)
	}

	defer func() {
		for i, container := range containers {
			if err := container.Terminate(ctx); err != nil {
				t.Logf("failed to terminate %s container: %v", services[i].name, err)
			}
		}
	}()

	client := &http.Client{Timeout: 5 * time.Second}
	for i, url := range urls {
		resp, err := client.Get(url)
		if err != nil {
			t.Fatalf("failed to GET %s: %v", url, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("failed to read response from %s: %v", url, err)
		}

		actual := string(body)
		if actual != services[i].expected {
			t.Errorf("%s server: expected %q, got %q", services[i].name, services[i].expected, actual)
		} else {
			t.Logf("%s server returned correct response: %q", services[i].name, actual)
		}
	}
}
