# Docker Support Design

**Date:** 2026-05-19
**Branch:** docker-support
**Target:** PR to t0mer/go-dnstester

## Goal

Add first-class Docker support so users can run dnstester without installing Go or Node.js. DNS servers and FQDNs configurable via environment variables for easy deployment. Config persists across restarts via a named volume.

## Files Added

```
Dockerfile
entrypoint.sh
docker-compose.yml
.env.example
```

`.env` added to `.gitignore`.

## Dockerfile

- Base: `debian:12-slim`
- Downloads pre-built binary from GitHub releases using `TARGETARCH` to select correct artifact (`linux-amd64` or `linux-arm64` or `linux-armhf`)
- Installs `curl` and `jq` (jq used by entrypoint for JSON generation)
- Copies `entrypoint.sh`, sets executable
- Exposes port `7020`
- Entrypoint: `entrypoint.sh`

Pinned to a specific release tag via `ARG VERSION=2026.5.1` (overridable at build time).

## entrypoint.sh

Runs at container startup. Logic:

1. If `/config/dnstester.json` already exists → skip config generation, exec binary directly
2. If not → parse env vars and write `/config/dnstester.json`:
   - `DNS_SERVERS=Name:IP,Name:IP,...` → `servers` array with `enabled: true`
   - `DNS_FQDNS=a.com,b.com,...` → `fqdns` array
   - If env vars absent → write nothing (binary uses its own built-in defaults)
3. Exec `dnstester --conf /config --port $DNS_PORT`

**First-run-only write** means UI edits persist across restarts. Env vars are a seed, not an override.

## docker-compose.yml

```yaml
services:
  dnstester:
    build: .
    ports:
      - "${DNS_PORT:-7020}:${DNS_PORT:-7020}"
    volumes:
      - dnstester-config:/config
    env_file: .env

volumes:
  dnstester-config:
```

## .env.example

```env
DNS_PORT=7020
DNS_SERVERS=Cloudflare:1.1.1.1,Google:8.8.8.8,Quad9:9.9.9.9
DNS_FQDNS=google.com,cloudflare.com,github.com
```

## Architecture Decisions

- **Pre-built binary over multi-stage build:** Keeps Dockerfile simple and fast. Build-from-source is available via `make build-local` for users who prefer it.
- **First-run-only config write:** Prevents env vars from wiping UI changes on every restart. Users who want to reset to env var defaults can delete the volume.
- **jq for JSON generation:** Avoids Python/node dep; `jq` is small and available in debian-slim via apt.
- **No new code paths in Go:** Purely infrastructure layer, zero changes to application code.

## Out of Scope

- Kubernetes / Helm charts
- Health check endpoint (app doesn't expose one)
- Multi-arch image push to Docker Hub (CI can be added later)
