#!/usr/bin/env bash
# scripts/build.sh — builds UI then cross-compiles the Go binary.
# Idempotent: safe to run repeatedly.

set -euo pipefail

VERSION="${VERSION:-dev}"
BUILD_MODE="${BUILD_MODE:-prod}"
UI_DIR="web/ui"
EMBED_DIR="web/dist"
OUT_DIR="dist"

# Format: GOOS/GOARCH/GOARM:output-suffix  (GOARM empty when unused)
TARGETS="${TARGETS:-
  linux/amd64/:linux-amd64
  linux/arm/7:linux-armhf
  linux/arm64/:linux-arm64
  linux/arm64/:linux-aarch64
  windows/amd64/:windows-amd64
}"

log() { printf '\033[1;34m[build]\033[0m %s\n' "$*"; }

build_ui() {
  log "Building UI ($UI_DIR)"
  pushd "$UI_DIR" >/dev/null
  if [ -f package-lock.json ]; then npm ci; else npm install; fi
  npm run build
  popd >/dev/null

  log "Staging UI assets → $EMBED_DIR"
  rm -rf "$EMBED_DIR"
  mkdir -p "$EMBED_DIR"

  if [ -d "$UI_DIR/dist" ]; then
    cp -R "$UI_DIR/dist/." "$EMBED_DIR/"
  else
    echo "UI build output not found under $UI_DIR/dist" >&2
    exit 1
  fi
}

build_go() {
  local spec="$1"                      # e.g. linux/arm/7:linux-armhf
  local target="${spec%%:*}"           # linux/arm/7
  local suffix="${spec##*:}"           # linux-armhf

  local goos goarch goarm ext=""
  IFS='/' read -r goos goarch goarm <<< "$target"

  [ "$goos" = "windows" ] && ext=".exe"

  local out="$OUT_DIR/dnstester-${VERSION}-${suffix}${ext}"
  log "Compiling $out"
  mkdir -p "$OUT_DIR"

  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" GOARM="${goarm:-}" \
    go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.buildMode=${BUILD_MODE}" \
      -o "$out" \
      ./cmd/dnstester
}

main() {
  log "Mode=$BUILD_MODE Version=$VERSION"
  build_ui

  if [ "$BUILD_MODE" = "local" ]; then
    CGO_ENABLED=0 go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.buildMode=${BUILD_MODE}" \
      -o "$OUT_DIR/dnstester-${VERSION}-$(go env GOOS)-$(go env GOARCH)" \
      ./cmd/dnstester
    return
  fi

  while IFS= read -r target; do
    target="${target//[[:space:]]/}"
    [ -z "$target" ] && continue
    build_go "$target"
  done <<< "$TARGETS"

  log "Done — artifacts in $OUT_DIR/"
}

main "$@"
