#!/usr/bin/env bash
# scripts/build.sh — builds UI then cross-compiles the Go binary.
# Idempotent: safe to run repeatedly.

set -euo pipefail

VERSION="${VERSION:-dev}"
BUILD_MODE="${BUILD_MODE:-prod}"
UI_DIR="web/ui"
EMBED_DIR="web/dist"
OUT_DIR="dist"

TARGETS="${TARGETS:-linux/amd64 linux/arm linux/arm64 linux/386 windows/amd64 windows/386 darwin/amd64 darwin/arm64}"

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

  # Vite outputs to web/ui/dist; copy everything into web/dist.
  if [ -d "$UI_DIR/dist" ]; then
    cp -R "$UI_DIR/dist/." "$EMBED_DIR/"
  else
    echo "UI build output not found under $UI_DIR/dist" >&2
    exit 1
  fi
}

build_go() {
  local goos="$1" goarch="$2"
  local ext=""
  [ "$goos" = "windows" ] && ext=".exe"
  local out="$OUT_DIR/dnstester-${VERSION}-${goos}-${goarch}${ext}"

  log "Compiling $out"
  mkdir -p "$OUT_DIR"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
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
    build_go "$(go env GOOS)" "$(go env GOARCH)"
    return
  fi

  for target in $TARGETS; do
    build_go "${target%/*}" "${target#*/}"
  done

  log "Done — artifacts in $OUT_DIR/"
}

main "$@"
