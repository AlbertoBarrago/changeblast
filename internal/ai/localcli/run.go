// Package localcli implements the ai.Provider contract against a
// locally installed, already-authenticated AI CLI (Claude Code, Codex,
// Gemini) instead of a raw provider API. This is deliberate: a user who
// already has one of these tools set up (logged in via its own
// subscription/account) gets --explain working with zero extra
// configuration — no API key to obtain, no environment variable to set.
// Serval never manages credentials for these tools; it only shells
// out to them the same way a user would from their own terminal.
package localcli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// requestTimeout bounds how long a single local-CLI explanation call
// may take. These invoke a full agent CLI, not a raw completion
// endpoint, so this is more generous than Ollama's direct-API timeout.
const requestTimeout = 180 * time.Second

// run executes binary with args (a lookPath check first, for a clear
// "not installed" error instead of a bare exec failure), capturing
// stdout and stderr separately. A non-zero exit returns an error
// including the tool's own stderr, so an unauthenticated or
// misconfigured CLI surfaces its actual diagnostic rather than a bare
// "exit status 1".
func run(ctx context.Context, providerName, binary string, args []string) (string, error) {
	if _, err := exec.LookPath(binary); err != nil {
		return "", fmt.Errorf("%s: %q not found on PATH — install it and sign in first: %w", providerName, binary, err)
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("%s: %s failed: %w: %s", providerName, binary, err, msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}
