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

types-generate:
    @tygo generate --config tygo.yml

types-check:
    @tygo generate --config tygo.yml
    @git diff --exit-code web/src/lib/types/generated/ || (echo "Generated types are out of date. Run 'just types-generate' and commit." && exit 1)

run-spa:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_DEV=1 ARABICA_SPA=1 go run ./cmd/arabica -known-dids known-dids.txt

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

# Run E2E tests with Playwright. Boots the e2e-server (test PDS + SPA),
# then runs the Playwright spec files against it.
e2e: e2e-build
    @echo 'Starting e2e-server...'
    @go run -tags=integration ./cmd/e2e-server & echo $$! > /tmp/e2e-server.pid
    @sleep 3
    @echo 'Running Playwright tests...'
    @cd web && CHROMIUM_PATH=$$(nix-shell -p chromium --run 'which chromium' 2>/dev/null | tail -1) pnpm exec playwright test
    @kill `cat /tmp/e2e-server.pid` 2>/dev/null || true

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
    @cd web && pnpm run test
    @pnpm run test:svelte
