arabica: templ-watch-arabica

oolong: templ-watch-oolong

run:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_DEV=1 go run ./cmd/arabica -known-dids known-dids.txt

run-oolong: templ-generate
    @LOG_LEVEL=debug LOG_FORMAT=console OOLONG_DEV=1 go run ./cmd/oolong

svelte-build:
    @pnpm run check:svelte
    @pnpm run build:svelte

spa-build:
    @./scripts/build-spa.sh

spa-dev:
    @cd web && pnpm run dev

# Watch web/ source and rebuild the SvelteKit SPA to web/build on every change.
# Used by run-spa-dev / run-oolong-spa-dev; run standalone to just rebuild on save.
spa-watch:
    @./scripts/watch-spa.sh

types-generate:
    @tygo generate --config tygo.yml

types-check:
    @tygo generate --config tygo.yml
    @git diff --exit-code web/src/lib/types/generated/ || (echo "Generated types are out of date. Run 'just types-generate' and commit." && exit 1)

run-spa:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_DEV=1 ARABICA_SPA=1 go run ./cmd/arabica -known-dids known-dids.txt

# Run Arabica with SPA dev hot-reload. scripts/watch-spa.sh rebuilds the
# SvelteKit bundle to web/build on every source change; ARABICA_DEV=1 makes
# the Go server re-read index.html and /_app/ chunks from disk on each
# request, so changes appear on the next browser refresh without restarting
# Go. Requires `pnpm` and `inotifywait` (inotify-tools).
run-spa-dev:
    @./scripts/watch-spa.sh & \
        trap 'kill $$!' EXIT; \
        LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_DEV=1 ARABICA_SPA=1 go run ./cmd/arabica -known-dids known-dids.txt

# Run Oolong with the embedded SvelteKit shell enabled. Only routes listed in
# Oolong's SPAOwnedRoutes are served by the SPA; all other pages stay legacy.
run-oolong-spa: spa-build
    @LOG_LEVEL=debug LOG_FORMAT=console OOLONG_DEV=1 OOLONG_SPA=1 go run ./cmd/oolong

# Run Oolong with SPA dev hot-reload (see run-spa-dev).
run-oolong-spa-dev:
    @./scripts/watch-spa.sh & \
        trap 'kill $$!' EXIT; \
        LOG_LEVEL=debug LOG_FORMAT=console OOLONG_DEV=1 OOLONG_SPA=1 go run ./cmd/oolong

build:
    @pnpm run build:svelte
    @./scripts/build-spa.sh
    @templ generate
    @go build ./cmd/arabica

templ-watch-arabica:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_DEV=1 templ generate --watch --proxy="http://localhost:18079" --cmd="go run ./cmd/arabica -known-dids known-dids.txt"

templ-watch-oolong:
    @LOG_LEVEL=debug LOG_FORMAT=console OOLONG_DEV=1 templ generate --watch --proxy="http://localhost:18081" --cmd="go run ./cmd/oolong"

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

# Run Playwright against the Oolong-configured test harness. Pass a spec or
# Playwright arguments after the target to narrow the currently ported flows.
e2e-oolong *args: e2e-build
    @ARABICA_E2E_APP=oolong just _e2e-run {{args}}

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

# Run only the Oolong E2E server for manual SPA testing.
e2e-oolong-server: e2e-build
    @ARABICA_E2E_APP=oolong go run -tags=integration ./cmd/e2e-server

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
