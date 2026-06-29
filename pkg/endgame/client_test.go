package endgame

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type captureRoundTripper struct {
	requests []*http.Request
}

func (c *captureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	c.requests = append(c.requests, req.Clone(req.Context()))
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`)),
		Request:    req,
	}, nil
}

func TestClientSendsConfiguredHeadersOnEveryCall(t *testing.T) {
	transport := &captureRoundTripper{}
	client, err := NewClient(&http.Client{Transport: transport}, WithHeaders(map[string]string{
		"X-Test-Header": "abc",
	}))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	if err := client.Initialize(); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	if _, err := client.ListTools(); err != nil {
		t.Fatalf("ListTools returned error: %v", err)
	}

	if len(transport.requests) != 2 {
		t.Fatalf("captured %d requests, want 2", len(transport.requests))
	}
	for i, req := range transport.requests {
		if got := req.Header.Get("X-Test-Header"); got != "abc" {
			t.Fatalf("request %d X-Test-Header = %q, want abc", i, got)
		}
	}
}

func TestClientManagedProtocolHeadersWin(t *testing.T) {
	transport := &captureRoundTripper{}
	client, err := NewClient(&http.Client{Transport: transport}, WithHeaders(map[string]string{
		"Content-Type": "text/plain",
		"Accept":       "text/plain",
	}))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	if err := client.Initialize(); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	req := transport.requests[0]
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := req.Header.Get("Accept"); got != "application/json, text/event-stream" {
		t.Fatalf("Accept = %q, want application/json, text/event-stream", got)
	}
}
