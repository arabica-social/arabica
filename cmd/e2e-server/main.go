// Command e2e-server boots the arabica handler tree (with the SvelteKit SPA
// shell enabled) backed by an in-process test PDS. It is used by the
// Playwright E2E test suite.
//
// The server writes its listen URL to tests/e2e/.server-url on startup so
// Playwright (or the justfile e2e target) can discover it.
//
// Usage:
//
//	just e2e
//
// Which runs:
//  1. Build the SvelteKit SPA (pnpm run build + build-spa.sh)
//  2. This server (boots, writes URL, serves until killed)
//  3. Playwright (reads URL, runs .spec.ts files, kills server on exit)
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"tangled.org/arabica.social/arabica/tests/integration"
)

const serverURLFile = "tests/e2e/.server-url"
const serverDIDFile = "tests/e2e/.server-did"

func main() {
	// The integration harness needs a *testing.T for t.Cleanup and
	// require helpers. We create a throwaway testing.T — the harness
	// registers cleanup on it, but we manage the lifecycle manually via
	// signal handling since this is a long-running process.
	t := &testing.T{}

	h := integration.StartHarness(t, &integration.HarnessOptions{
		EnableFirehose: true,
		EnableSPA:      true,
	})

	// Write the server URL for Playwright to read.
	if err := os.MkdirAll("tests/e2e", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create e2e dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(serverURLFile, []byte(h.Server.URL), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write server URL: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(serverURLFile)

	// Write the primary account DID for Playwright fixtures.
	if err := os.WriteFile(serverDIDFile, []byte(h.PrimaryAccount.DID), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write server DID: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(serverDIDFile)

	fmt.Printf("E2E server running at %s\n", h.Server.URL)
	fmt.Printf("  Primary account: DID=%s, Handle=%s\n", h.PrimaryAccount.DID, h.PrimaryAccount.Handle)
	fmt.Printf("  Server URL written to %s\n", serverURLFile)
	fmt.Printf("  Press Ctrl+C to stop.\n")

	// Wait for a signal to shut down.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	fmt.Println("\nShutting down E2E server...")
	h.Server.Close()

	// Give in-flight requests a moment to complete.
	time.Sleep(200 * time.Millisecond)
}
