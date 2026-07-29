#!/usr/bin/env bash

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
