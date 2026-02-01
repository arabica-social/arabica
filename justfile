[private]
default: style templ-generate run

run:
    @LOG_LEVEL=debug LOG_FORMAT=console go run cmd/server/main.go -known-dids known-dids.txt

templ-watch:
    @templ generate --watch --proxy="http://localhost:18080" --cmd="go run ./cmd/server -known-dids known-dids.txt"

templ-generate:
    @templ generate

test:
    @go test ./... -cover -coverprofile=cover.out

style:
    @nix develop --command tailwindcss -i web/static/css/app.css -o web/static/css/output.css --minify
