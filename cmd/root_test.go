package cmd

import (
	"testing"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/endgame"
)

func TestConfigureMCPHeadersFromImpersonateOrg(t *testing.T) {
	old := impersonateOrgID
	defer func() {
		impersonateOrgID = old
		auth.SetConfiguredMCPHeaders(nil)
	}()

	impersonateOrgID = " 6054 "
	if err := configureMCPHeaders(); err != nil {
		t.Fatalf("configureMCPHeaders returned error: %v", err)
	}

	headers := auth.ConfiguredMCPHeaders()
	if got := headers[impersonateOrgHeader]; got != "6054" {
		t.Fatalf("%s = %q, want 6054", impersonateOrgHeader, got)
	}
}

func TestConfigureMCPEndpointFromPreview(t *testing.T) {
	old := previewPullRequest
	defer func() {
		previewPullRequest = old
		auth.SetConfiguredMCPEndpoint("")
	}()

	previewPullRequest = 12603
	if err := configureMCPEndpoint(); err != nil {
		t.Fatalf("configureMCPEndpoint returned error: %v", err)
	}

	want, err := endgame.PreviewURL(12603)
	if err != nil {
		t.Fatalf("PreviewURL returned error: %v", err)
	}
	if got := auth.ConfiguredMCPEndpoint(); got != want {
		t.Fatalf("ConfiguredMCPEndpoint = %q, want %q", got, want)
	}
}

func TestConfigureMCPEndpointClearsProductionDefault(t *testing.T) {
	old := previewPullRequest
	defer func() {
		previewPullRequest = old
		auth.SetConfiguredMCPEndpoint("")
	}()

	auth.SetConfiguredMCPEndpoint("https://preview.example.test/api/v1/mcp")
	previewPullRequest = 0
	if err := configureMCPEndpoint(); err != nil {
		t.Fatalf("configureMCPEndpoint returned error: %v", err)
	}

	if got := auth.ConfiguredMCPEndpoint(); got != "" {
		t.Fatalf("ConfiguredMCPEndpoint = %q, want empty", got)
	}
}

func TestConfigureMCPEndpointRejectsNegativePreview(t *testing.T) {
	old := previewPullRequest
	defer func() {
		previewPullRequest = old
		auth.SetConfiguredMCPEndpoint("")
	}()

	previewPullRequest = -1
	if err := configureMCPEndpoint(); err == nil {
		t.Fatal("configureMCPEndpoint returned nil error")
	}
}

func TestConfigureMCPHeadersClearsEmptyImpersonateOrg(t *testing.T) {
	old := impersonateOrgID
	defer func() {
		impersonateOrgID = old
		auth.SetConfiguredMCPHeaders(nil)
	}()

	auth.SetConfiguredMCPHeaders(map[string]string{impersonateOrgHeader: "6054"})
	impersonateOrgID = " "
	if err := configureMCPHeaders(); err != nil {
		t.Fatalf("configureMCPHeaders returned error: %v", err)
	}

	if headers := auth.ConfiguredMCPHeaders(); len(headers) != 0 {
		t.Fatalf("ConfiguredMCPHeaders = %#v, want empty", headers)
	}
}
