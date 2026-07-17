//go:build integration

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
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"tangled.org/arabica.social/arabica/tests/integration"
)

const serverURLFile = "tests/e2e/.server-url"
const serverDIDFile = "tests/e2e/.server-did"
const controlURLFile = "tests/e2e/.control-url"

var accountSequence atomic.Uint64

func main() {
	app := flag.String("app", e2eAppFromEnv(), "app to serve (defaults to arabica)")
	flag.Parse()

	dataDir, err := os.MkdirTemp("", "arabica-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create e2e data directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dataDir)

	h, err := integration.StartHarnessRuntime(context.Background(), filepath.Join(dataDir, "harness"), &integration.HarnessOptions{
		App:            *app,
		EnableFirehose: true,
		EnableSPA:      true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start e2e harness: %v\n", err)
		os.Exit(1)
	}
	defer h.Close()

	controlServer := httptest.NewServer(e2eControlHandler(h))
	defer controlServer.Close()

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

	if err := os.WriteFile(controlURLFile, []byte(controlServer.URL), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write control server URL: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(controlURLFile)

	fmt.Printf("E2E server running at %s\n", h.Server.URL)
	fmt.Printf("  Primary account: DID=%s, Handle=%s\n", h.PrimaryAccount.DID, h.PrimaryAccount.Handle)
	fmt.Printf("  Server URL written to %s\n", serverURLFile)
	fmt.Printf("  Control URL written to %s\n", controlURLFile)
	fmt.Printf("  Press Ctrl+C to stop.\n")

	// Wait for a signal to shut down.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	fmt.Println("\nShutting down E2E server...")
	// Give in-flight requests a moment to complete.
	time.Sleep(200 * time.Millisecond)
}

func e2eAppFromEnv() string {
	if app := os.Getenv("ARABICA_E2E_APP"); app != "" {
		return app
	}
	return "arabica"
}

func e2eControlHandler(h *integration.Harness) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts", func(w http.ResponseWriter, r *http.Request) {
		id := accountSequence.Add(1)
		handle := fmt.Sprintf("e2e%d.test", id)
		account, err := h.CreateRuntimeAccount(
			fmt.Sprintf("e2e%d@test.com", id),
			handle,
			"e2e-password",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeControlJSON(w, http.StatusCreated, map[string]string{
			"did":        account.DID,
			"handle":     account.Handle,
			"session_id": h.SessionIDFor(account),
		})
	})
	mux.HandleFunc("GET /wait-index", func(w http.ResponseWriter, r *http.Request) {
		uri := r.URL.Query().Get("uri")
		if uri == "" {
			http.Error(w, "uri is required", http.StatusBadRequest)
			return
		}
		present, err := strconv.ParseBool(r.URL.Query().Get("present"))
		if err != nil {
			http.Error(w, "present must be true or false", http.StatusBadRequest)
			return
		}
		if err := h.WaitForRecordState(r.Context(), uri, present, 10*time.Second); err != nil {
			http.Error(w, err.Error(), http.StatusGatewayTimeout)
			return
		}
		writeControlJSON(w, http.StatusOK, map[string]bool{"ready": true})
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeControlJSON(w, http.StatusOK, map[string]bool{"ready": true})
	})
	return mux
}

func writeControlJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
