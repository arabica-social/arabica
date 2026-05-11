[private]
default: templ-generate run

# Run dev server with CSS hot-reload — edit any file under
# internal/web/assets/css/ and refresh; no rebuild needed.
run:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_CSS_HOTRELOAD=1 go run ./cmd/server -known-dids known-dids.txt

templ-watch:
    @templ generate --watch --proxy="http://localhost:18080" --cmd="ARABICA_CSS_HOTRELOAD=1 go run ./cmd/server -known-dids known-dids.txt"

templ-generate:
    @templ generate

test:
    @templ generate
    @go test ./... -cover -coverprofile=cover.out

integration-test:
    @cd tests/integration && go test -v ./... -count=1

verbose-integration-test:
    @cd tests/integration && INTEGRATION_LOGS=true go test -v ./... -count=1
