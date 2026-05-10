[private]
default: style templ-generate run

run:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json go run ./cmd/server -known-dids known-dids.txt

templ-watch:
    @templ generate --watch --proxy="http://localhost:18080" --cmd="go run ./cmd/server -known-dids known-dids.txt"

templ-generate:
    @templ generate

test:
    @templ generate
    @go test ./... -cover -coverprofile=cover.out

integration-test:
    @cd tests/integration && go test -v ./... -count=1 

verbose-integration-test:
    @cd tests/integration && INTEGRATION_LOGS=true go test -v ./... -count=1 

style: style-arabica

# style-oolong

style-arabica:
    @nix develop --command tailwindcss -i static/css/app.css -o static/css/output.css --minify

style-oolong:
    @nix develop --command tailwindcss -i static/css/app-oolong.css -o static/css/output-oolong.css --minify
