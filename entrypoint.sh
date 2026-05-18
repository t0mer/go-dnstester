#!/usr/bin/env sh
set -e

CONFIG_FILE="/config/dnstester.json"
PORT="${DNS_PORT:-7020}"

if [ -f "$CONFIG_FILE" ]; then
    exec /usr/local/bin/dnstester --conf /config --port "$PORT"
fi

# First run: seed config from env vars if provided
if [ -n "$DNS_SERVERS" ] || [ -n "$DNS_FQDNS" ]; then
    mkdir -p /config

    servers_json='[]'
    if [ -n "$DNS_SERVERS" ]; then
        servers_json=$(printf '%s' "$DNS_SERVERS" | tr ',' '\n' | \
            jq -Rn '[inputs | split(":") | {"name": .[0], "address": .[1], "enabled": true}]')
    fi

    fqdns_json='[]'
    if [ -n "$DNS_FQDNS" ]; then
        fqdns_json=$(printf '%s' "$DNS_FQDNS" | tr ',' '\n' | jq -Rn '[inputs]')
    fi

    jq -n \
        --argjson servers "$servers_json" \
        --argjson fqdns "$fqdns_json" \
        '{"servers": $servers, "fqdns": $fqdns, "schedules": []}' \
        > "$CONFIG_FILE"
fi

exec /usr/local/bin/dnstester --conf /config --port "$PORT"
