#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Building SvelteKit SPA..."
cd "$ROOT_DIR/web"
pnpm run build

echo "Copying build output to embed directory..."
BUILD_DIR="$ROOT_DIR/internal/web/spa/build"
mkdir -p "$BUILD_DIR"
# Preserve the tracked marker while replacing generated files.
find "$BUILD_DIR" -mindepth 1 -maxdepth 1 ! -name .gitkeep -exec rm -rf {} +
cp -r "$ROOT_DIR/web/build/"* "$BUILD_DIR/"

echo "SPA build complete."
