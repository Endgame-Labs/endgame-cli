package cmd

import (
	"errors"
	"io"
	"os"
	"strings"
)

func resolvePrompt(flagValue string) (string, error) {
	if trimmed := strings.TrimSpace(flagValue); trimmed != "" {
		return trimmed, nil
	}

	info, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return "", errors.New("prompt required: pass --prompt or pipe stdin")
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}

	prompt := strings.TrimSpace(string(data))
	if prompt == "" {
		return "", errors.New("prompt required: stdin was empty")
	}

	return prompt, nil
}
