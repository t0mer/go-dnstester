FROM debian:12-slim

ARG VERSION=2026.5.1
ARG TARGETARCH
ARG TARGETVARIANT

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl jq && rm -rf /var/lib/apt/lists/*

RUN set -e; \
    case "${TARGETARCH}" in \
        amd64) SUFFIX="linux-amd64" ;; \
        arm64) SUFFIX="linux-arm64" ;; \
        arm)   SUFFIX="linux-armhf" ;; \
        *) echo "Unsupported arch: ${TARGETARCH}" && exit 1 ;; \
    esac; \
    curl -fsSL \
        "https://github.com/t0mer/go-dnstester/releases/download/${VERSION}/dnstester-${VERSION}-${SUFFIX}" \
        -o /usr/local/bin/dnstester && \
    chmod +x /usr/local/bin/dnstester

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 7020

ENTRYPOINT ["/entrypoint.sh"]
