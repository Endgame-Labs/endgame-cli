package auth

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
	"golang.org/x/oauth2"
)

type staticTokenSource struct {
	token *oauth2.Token
}

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	return s.token, nil
}

type recordingRoundTripper struct {
	requests []*http.Request
}

func (r *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r.requests = append(r.requests, req.Clone(req.Context()))
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`)),
		Request:    req,
	}, nil
}

func TestConfiguredAuthorizationHeaderCannotOverrideOAuth(t *testing.T) {
	recorder := &recordingRoundTripper{}
	httpClient := &http.Client{
		Transport: &authTransport{
			Base: recorder,
			Source: staticTokenSource{token: &oauth2.Token{
				AccessToken: "real",
				TokenType:   "Bearer",
				Expiry:      time.Now().Add(time.Hour),
			}},
		},
	}
	client, err := endgame.NewClient(httpClient, endgame.WithHeaders(map[string]string{
		"Authorization":           "Bearer wrong",
		"X-Endgame-Act-As-Org-Id": "6054",
		"Mcp-Session-Id":          "wrong-session",
		"Content-Type":            "text/plain",
		"Accept":                  "text/plain",
	}))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	if err := client.Initialize(); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	req := recorder.requests[0]
	if got := req.Header.Get("Authorization"); got != "Bearer real" {
		t.Fatalf("Authorization = %q, want Bearer real", got)
	}
	if got := req.Header.Get("X-Endgame-Act-As-Org-Id"); got != "6054" {
		t.Fatalf("X-Endgame-Act-As-Org-Id = %q, want 6054", got)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}
