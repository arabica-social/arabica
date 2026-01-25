run:
    @LOG_LEVEL=debug LOG_FORMAT=console go run cmd/arabica-server/main.go -known-dids known-dids.txt

run-production:
    @LOG_FORMAT=json SECURE_COOKIES=true go run cmd/arabica-server/main.go

test:
    @go test ./... -cover -coverprofile=cover.out

style:
    @nix develop --command tailwindcss -i static/css/style.css -o static/css/output.css --minify

build-ui:
    @pushd frontend || exit 1 && npm run build && popd || exit 1
