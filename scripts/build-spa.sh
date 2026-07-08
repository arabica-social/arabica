#!/usr/bin/env bash
# Build the SvelteKit SPA and copy output into the Go embed directory.
#
# The SvelteKit app lives in web/ and builds to web/build/. The Go server
# embeds internal/web/spa/build/ via go:embed, so the build output must be
# copied there before `go build`.
#
# Usage: ./scripts/build-spa.sh
# Or via justfile: just build-spa

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Building SvelteKit SPA..."
cd "$ROOT_DIR/web"
pnpm run build

echo "Copying build output to embed directory..."
rm -rf "$ROOT_DIR/internal/web/spa/build"
mkdir -p "$ROOT_DIR/internal/web/spa/build"
cp -r "$ROOT_DIR/web/build/"* "$ROOT_DIR/internal/web/spa/build/"

echo "SPA build complete."
