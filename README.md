# go-dnstester

A single Go binary that benchmarks public DNS servers by querying a configurable list of FQDNs, records response times, pings each server, and presents the results in a web UI.

## Features

### Results Table
Query results displayed in a sortable, filterable table showing each DNS server, FQDN queried, response time (ms), status, and resolved answer.

![Results Table](screenshots/results_table.png)

### Response Time Graph
Bar chart of average response time per DNS server for the current test run.

![Results Graph](screenshots/results_graph.png)

### Ping Results
ICMP ping latency to each configured DNS server, shown alongside DNS query results.

![Ping Results](screenshots/ping_results.png)

### Test History
Full log of every test run — timestamp, query count, success rate, and average response time. Load any past run into the results view or set it as a baseline for comparison.

![Test History](screenshots/tests_history.png)

### Run Comparison
Select any two historical runs (A = baseline, B = comparison) and see an overall delta, a side-by-side bar chart, and a per-server breakdown with percentage change.

![Compare Results](screenshots/compare_results.png)

### Settings & Scheduled Scans
Manage DNS servers and FQDNs. Backup, restore, export, and import configuration. Create scheduled scans that run automatically on a cron interval — results are tagged and available in History and Compare.

![Settings](screenshots/settings.png)

## Getting Started

### Prerequisites

- Go 1.21+
- Node.js 18+ (for UI development)

### Run

```bash
make build-local
./dist/dnstester-<version>-linux-amd64
```

Then open [http://localhost:7020](http://localhost:7020) in your browser.

## Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `7020` | Port for the web UI and API |
| `--conf` | *(see below)* | Path to the config directory |

### Config directory resolution (in order of precedence)

1. `--conf <path>` — CLI flag wins when provided
2. `CONFIG_PATH` environment variable — used when `--conf` is absent
3. OS default — `$XDG_CONFIG_HOME/dnstester` (Linux) / `~/Library/Application Support/dnstester` (macOS)

```bash
# Use a custom config directory via flag
./dnstester --conf /etc/dnstester

# Use environment variable
CONFIG_PATH=/etc/dnstester ./dnstester

# Flag beats env var when both are set
CONFIG_PATH=/etc/dnstester ./dnstester --conf /opt/dnstester  # uses /opt/dnstester
```

## Default Configuration

**DNS Servers**

| Name | Address |
|------|---------|
| Cloudflare | 1.1.1.1 |
| Cloudflare Alt | 1.0.0.1 |
| Google | 8.8.8.8 |
| Google Alt | 8.8.4.4 |
| Quad9 | 9.9.9.9 |
| OpenDNS | 208.67.222.222 |
| OpenDNS Alt | 208.67.220.220 |
| AdGuard | 94.140.14.14 |

**FQDNs queried by default**

`google.com` · `cloudflare.com` · `github.com` · `microsoft.com` · `apple.com`

All servers and FQDNs are configurable from the Settings page or via the API.

## Build Commands

```bash
make build             # Cross-compile production binary
make build-dev         # Cross-compile dev binary
make build-local       # Build for local architecture only (fastest)
make test              # go test ./...
make lint              # go vet ./...
make clean             # Remove build artifacts and dist/
make package VERSION=1.0.0       # Build production .ipk package
make package-dev VERSION=1.0.0   # Build dev .ipk package
```

Run a single test:

```bash
go test ./internal/service/... -run TestDNSQuery -v
```

UI dev server (proxies API to Go server on `:7020`):

```bash
cd web/ui && npm run dev
```

## REST API

The web UI is backed by a REST API. An interactive Swagger UI is available at `/api/docs` and the OpenAPI spec at `/api/openapi.json`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/test/run` | Run a DNS test and return results |
| `POST` | `/api/test/run` | Same as GET (for clients that prefer POST) |
| `GET` | `/api/test/latest` | Return the most recent test results |
| `GET` | `/api/history` | List all historical test runs |
| `GET` | `/api/history/{id}` | Get a specific historical run |
| `GET` | `/api/compare?a={id}&b={id}` | Compare two historical runs |
| `GET` | `/api/settings` | Get current configuration |
| `PUT` | `/api/settings` | Replace current configuration |
| `POST` | `/api/config/backup` | Create a config backup |
| `POST` | `/api/config/restore` | Restore config from backup |
| `GET` | `/api/config/export` | Download config as JSON |
| `POST` | `/api/config/import` | Import config from JSON |
| `GET` | `/api/schedules` | List all scheduled scans |
| `POST` | `/api/schedules` | Create a scheduled scan |
| `PUT` | `/api/schedules/{id}` | Update a scheduled scan |
| `DELETE` | `/api/schedules/{id}` | Delete a scheduled scan |
| `GET` | `/metrics` | Prometheus metrics endpoint |

## Prometheus Metrics

The `/metrics` endpoint exposes Prometheus-compatible metrics including DNS query response times and the results of the last 5 test runs, suitable for scraping by a Prometheus server or Grafana agent.

## License

See [LICENSE](LICENSE).
