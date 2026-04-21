package cmd

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Endgame-Labs/endgame-cli/pkg/auth"
	"github.com/Endgame-Labs/endgame-cli/pkg/toolscache"
	"github.com/spf13/cobra"
)

const (
	autoSyncEnvVar = "ENDGAME_SKIP_AUTO_TOOL_SYNC"
	autoSyncMaxAge = 12 * time.Hour
)

func maybeStartAutoToolSync(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	if strings.TrimSpace(os.Getenv(autoSyncEnvVar)) == "1" {
		return
	}
	if !isAuthenticatedForAutoSync() {
		return
	}
	if !isToolCacheStaleForAutoSync() {
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		return
	}

	bg := exec.Command(exePath, "tools", "sync", "--quiet")
	bg.Env = append(os.Environ(), autoSyncEnvVar+"=1")
	bg.Stdout = io.Discard
	bg.Stderr = io.Discard
	bg.Stdin = nil
	_ = bg.Start()
}

func isAuthenticatedForAutoSync() bool {
	_, err := auth.Load()
	return err == nil
}

func isToolCacheStaleForAutoSync() bool {
	cache, err := toolscache.Load()
	if err != nil {
		return true
	}
	return toolscache.IsStale(cache, autoSyncMaxAge)
}
