package cmd

import (
	"testing"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
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
