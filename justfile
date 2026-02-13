[private]
default: style templ-generate run

run:
    @LOG_LEVEL=debug LOG_FORMAT=console ARABICA_MODERATORS_CONFIG=roles.json go run cmd/server/main.go -known-dids known-dids.txt
    # @bash scripts/run.sh

templ-watch:
    @templ generate --watch --proxy="http://localhost:18080" --cmd="go run ./cmd/server -known-dids known-dids.txt"

templ-generate:
    @templ generate

test:
    @templ generate
    @go test ./... -cover -coverprofile=cover.out

style:
    @nix develop --command tailwindcss -i static/css/app.css -o static/css/output.css --minify
