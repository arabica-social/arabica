[private]
default: build-ui run

run:
    @LOG_LEVEL=debug LOG_FORMAT=console go run cmd/arabica-server/main.go -known-dids known-dids.txt

build-ui:
    @pushd frontend || exit 1 && pnpm run build && popd || exit 1

test:
    @go test ./... -cover -coverprofile=cover.out
