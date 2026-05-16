arabica: templ-watch-arabica

oolong: templ-watch-oolong

run:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_HOTRELOAD=1 go run ./cmd/arabica -known-dids known-dids.txt

run-oolong: templ-generate
    @LOG_LEVEL=debug LOG_FORMAT=console OOLONG_HOTRELOAD=1 go run ./cmd/oolong

templ-watch-arabica:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json ARABICA_HOTRELOAD=1 templ generate --watch --proxy="http://localhost:18080" --cmd="go run ./cmd/arabica -known-dids known-dids.txt"

templ-watch-oolong:
    @LOG_LEVEL=debug LOG_FORMAT=console OOLONG_HOTRELOAD=1 templ generate --watch --proxy="http://localhost:18080" --cmd="go run ./cmd/oolong"

templ-generate:
    @templ generate

test:
    @templ generate
    @go test ./... -cover -coverprofile=cover.out

integration-test:
    @cd tests/integration && go test -v ./... -count=1

verbose-integration-test:
    @cd tests/integration && INTEGRATION_LOGS=true go test -v ./... -count=1
