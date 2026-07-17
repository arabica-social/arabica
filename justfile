arabica: templ-watch-arabica

run:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_DEV=1 go run ./cmd/arabica -known-dids known-dids.txt

svelte-build:
    @pnpm run check:svelte
    @pnpm run build:svelte

spa-build:
    @./scripts/build-spa.sh

# Start only Vite's HMR server for isolated frontend work. It does not render
# Go's injected document head, so use `run-spa-dev` for normal local work.
spa-dev:
    @VITE_BACKEND_URL=http://127.0.0.1:18910 VITE_DEV_PORT=5173 pnpm --dir web run dev

# Legacy rebuild-on-refresh loop for exercising Go's disk-backed SPA shell.
# Use run-spa-dev for normal Vite HMR development.
spa-watch:
    @./scripts/watch-spa.sh

types-generate:
    @tygo generate --config tygo.yml

types-check:
    @tygo generate --config tygo.yml
    @git diff --exit-code web/src/lib/types/generated/ || (echo "Generated types are out of date. Run 'just types-generate' and commit." && exit 1)

run-spa:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_DEV=1 ARABICA_SPA=1 go run ./cmd/arabica -known-dids known-dids.txt

# Run Arabica's Go backend and disk-backed SPA rebuild watcher together. Open
# http://127.0.0.1:18910; refresh after a successful rebuild. OAuth returns to
# Go's browser-facing development origin, which serves the app's CSS and assets.
run-spa-dev:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_DEV=1 ARABICA_SPA=1 ARABICA_OAUTH_REDIRECT_URI=http://127.0.0.1:18910/oauth/callback go run ./cmd/arabica -known-dids known-dids.txt & \
        backend_pid=$$!; \
        ./scripts/watch-spa.sh & \
        watcher_pid=$$!; \
        trap 'kill $$watcher_pid $$backend_pid 2>/dev/null || true; wait $$watcher_pid 2>/dev/null || true; wait $$backend_pid 2>/dev/null || true' EXIT INT TERM; \
        wait $$watcher_pid

# Launch the Go backend and SPA rebuild watcher in a new tab in the current
# Herdr workspace. Pass workspace=true for a dedicated persistent workspace
# (which `just herdr-spa-dev-stop` closes). Outside Herdr, a dedicated
# workspace is created automatically.
herdr-spa-dev workspace='false':
    #!/usr/bin/env bash
    set -euo pipefail
    command -v herdr >/dev/null || { echo "error: herdr is required" >&2; exit 1; }
    command -v jq >/dev/null || { echo "error: jq is required" >&2; exit 1; }
    create_workspace="{{workspace}}"
    case "$create_workspace" in
        true|workspace=true) create_workspace=true ;;
        false|workspace=false) create_workspace=false ;;
        *) echo "error: workspace must be true or false" >&2; exit 2 ;;
    esac
    workspace_label="arabica spa dev"
    if [[ "$create_workspace" == "true" || "${HERDR_ENV:-}" != "1" ]]; then
        herdr workspace list | jq -r --arg label "$workspace_label" '.result.workspaces[] | select(.label == $label) | .workspace_id' | while IFS= read -r workspace_id; do
            [[ -z "$workspace_id" ]] || herdr workspace close "$workspace_id" >/dev/null
        done
        workspace_json="$(herdr workspace create --cwd "$PWD" --label "$workspace_label" --focus)"
        workspace_id="$(jq -er '.result.workspace.workspace_id' <<<"$workspace_json")"
        backend_pane="$(jq -er '.result.root_pane.pane_id' <<<"$workspace_json")"
    else
        tab_json="$(herdr tab create --workspace "$HERDR_WORKSPACE_ID" --cwd "$PWD" --label "Arabica SPA dev" --focus)"
        backend_pane="$(jq -er '.result.root_pane.pane_id' <<<"$tab_json")"
        workspace_id=""
    fi
    herdr pane rename "$backend_pane" "Arabica backend" >/dev/null
    herdr pane run "$backend_pane" 'exec env LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_DEV=1 ARABICA_SPA=1 ARABICA_OAUTH_REDIRECT_URI=http://127.0.0.1:18910/oauth/callback go run ./cmd/arabica -known-dids known-dids.txt' >/dev/null
    watcher_pane="$(herdr pane split "$backend_pane" --direction right --cwd "$PWD" --no-focus | jq -er '.result.pane.pane_id')"
    herdr pane rename "$watcher_pane" "SPA rebuilds" >/dev/null
    herdr pane run "$watcher_pane" 'exec ./scripts/watch-spa.sh' >/dev/null
    if [[ -n "$workspace_id" && "${HERDR_ENV:-}" == "1" ]]; then
        echo "Started Herdr workspace '$workspace_label' ($workspace_id); switch to it to view the backend and rebuild panes."
    elif [[ -n "$workspace_id" ]]; then
        exec herdr
    else
        echo "Started Arabica backend and SPA rebuild panes in a new Herdr tab."
    fi

herdr-spa-dev-stop:
    #!/usr/bin/env bash
    set -euo pipefail
    command -v herdr >/dev/null || { echo "error: herdr is required" >&2; exit 1; }
    command -v jq >/dev/null || { echo "error: jq is required" >&2; exit 1; }
    herdr workspace list | jq -r --arg label "arabica spa dev" '.result.workspaces[] | select(.label == $label) | .workspace_id' | while IFS= read -r workspace_id; do
        [[ -z "$workspace_id" ]] || herdr workspace close "$workspace_id" >/dev/null
    done

build:
    @pnpm run build:svelte
    @./scripts/build-spa.sh
    @templ generate
    @go build ./cmd/arabica

templ-watch-arabica:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_DEV=1 templ generate --watch --proxy="http://localhost:18079" --cmd="go run ./cmd/arabica -known-dids known-dids.txt"

templ-generate:
    @templ generate

test:
    @pnpm run build:svelte
    @./scripts/build-spa.sh
    @templ generate
    @go test ./... -cover -coverprofile=cover.out

integration-test:
    @cd tests/integration && go test -tags=integration -v ./... -count=1

verbose-integration-test:
    @cd tests/integration && INTEGRATION_LOGS=true go test -tags=integration -v ./... -count=1

format:
    @pnpm run format
    @gofmt -w ./
    @templ fmt . -w

# Build the SPA (needed before running e2e-server).
e2e-build:
    @pnpm run build:svelte
    @./scripts/build-spa.sh

# Boot the e2e-server (test PDS + SPA) and run Playwright with the given args.
# Private helper; call via `e2e` or `e2e-update-snapshots`.
_e2e-run *args:
    @set -eu; \
        rm -f tests/e2e/.server-url tests/e2e/.server-did tests/e2e/.control-url; \
        server_bin=$(mktemp /tmp/arabica-e2e-server.XXXXXX); \
        go build -tags=integration -o $server_bin ./cmd/e2e-server; \
        echo 'Starting e2e-server...'; \
        $server_bin & server_pid=$!; \
        trap 'kill $server_pid 2>/dev/null || true; wait $server_pid 2>/dev/null || true; rm -f $server_bin' EXIT INT TERM; \
        for attempt in $(seq 1 60); do \
            test -s tests/e2e/.server-url && test -s tests/e2e/.control-url && break; \
            kill -0 $server_pid 2>/dev/null || { echo 'e2e-server exited before becoming ready' >&2; exit 1; }; \
            sleep 0.25; \
        done; \
        test -s tests/e2e/.server-url && test -s tests/e2e/.control-url || { echo 'timed out waiting for e2e-server' >&2; exit 1; }; \
        echo 'Running Playwright tests...'; \
        base_url=$(cat tests/e2e/.server-url); \
        control_url=$(cat tests/e2e/.control-url); \
        cd web; \
        ARABICA_E2E_BASE_URL=$base_url ARABICA_E2E_CONTROL_URL=$control_url CHROMIUM_PATH=$(nix-shell -p chromium --run 'which chromium' 2>/dev/null | tail -1) pnpm exec playwright test {{args}}

# Run E2E tests with Playwright. Boots the e2e-server (test PDS + SPA),
# then runs the Playwright spec files against it.
e2e: e2e-build
    @just _e2e-run

# Update Playwright screenshot baselines after intentional UI changes.
# Defaults to the visual-regression spec; pass a spec + extra Playwright args
# to scope the update, e.g. after moving a single button:
#   just e2e-update-snapshots
#   just e2e-update-snapshots tests/e2e/visual-regression.spec.ts -g "brew view"
# Updated baselines land in web/tests/e2e/visual-regression.spec.ts-snapshots/.
e2e-update-snapshots testfile='tests/e2e/visual-regression.spec.ts' *args='': e2e-build
    @just _e2e-run {{testfile}} --update-snapshots {{args}}

# Run only the e2e-server (without Playwright) for manual testing.
e2e-server: e2e-build
    @go run -tags=integration ./cmd/e2e-server

# Run all CI checks locally (mirrors .github/workflows/ci.yml).
ci-check:
    @just types-check
    @templ generate
    @go vet ./...
    @go build ./cmd/arabica
    @go test ./... -count=1
    @cd tests/integration && go test -tags=integration -count=1 ./...
    @pnpm run check:svelte
    @cd web && pnpm run check
    @cd web && pnpm run test
    @pnpm run test:svelte
