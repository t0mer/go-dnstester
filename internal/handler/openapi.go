package handler

// openapiSpec is the OpenAPI 3.0 specification for the DNS Tester API.
const openapiSpec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "DNS Tester API",
    "description": "DNS performance testing: run ad-hoc or scheduled tests, browse history, view trends, and manage settings.",
    "version": "2.0.0"
  },
  "servers": [
    { "url": "/api", "description": "REST API" },
    { "url": "/",    "description": "Root (Prometheus metrics)" }
  ],
  "tags": [
    { "name": "Tests",    "description": "Run DNS tests and retrieve results" },
    { "name": "History",  "description": "Browse and compare historical test runs" },
    { "name": "Trends",   "description": "Aggregated response-time trends over time" },
    { "name": "Settings", "description": "Manage DNS servers, FQDNs, schedules, and global config" },
    { "name": "Updates",  "description": "Version info and in-app self-update" },
    { "name": "Monitoring","description": "Prometheus metrics export" }
  ],
  "paths": {
    "/metrics": {
      "get": {
        "summary": "Prometheus metrics",
        "description": "Exposes metrics in Prometheus text format (served at root, not under /api).",
        "operationId": "getMetrics",
        "tags": ["Monitoring"],
        "servers": [{ "url": "/" }],
        "responses": {
          "200": {
            "description": "Prometheus text exposition format",
            "content": { "text/plain": { "schema": { "type": "string" } } }
          }
        }
      }
    },

    "/test/run": {
      "get": {
        "summary": "Run a DNS test",
        "description": "Executes a DNS test against all enabled servers and FQDNs, persists the run, and returns the full result.",
        "operationId": "runTest",
        "tags": ["Tests"],
        "responses": {
          "200": { "description": "Test completed", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/TestRun" } } } },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      },
      "post": {
        "summary": "Run a DNS test (POST)",
        "description": "Identical to GET /test/run. For clients that prefer POST for non-idempotent operations.",
        "operationId": "runTestPost",
        "tags": ["Tests"],
        "responses": {
          "200": { "description": "Test completed", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/TestRun" } } } },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/test/latest": {
      "get": {
        "summary": "Get the latest test run",
        "description": "Returns the most recent test run (from memory if available, otherwise from the database).",
        "operationId": "getLatestTest",
        "tags": ["Tests"],
        "responses": {
          "200": { "description": "Latest test run", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/TestRun" } } } },
          "404": { "description": "No results yet", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },

    "/history": {
      "get": {
        "summary": "List test run history (paginated)",
        "description": "Returns a paginated list of run summaries. Defaults to the last 24 hours. Use offset+limit for paging.",
        "operationId": "listHistory",
        "tags": ["History"],
        "parameters": [
          { "name": "limit",     "in": "query", "description": "Results per page (default 100, max 500)",     "schema": { "type": "integer", "default": 100, "minimum": 1, "maximum": 500 } },
          { "name": "offset",    "in": "query", "description": "Number of rows to skip (for paging)",         "schema": { "type": "integer", "default": 0, "minimum": 0 } },
          { "name": "hours",     "in": "query", "description": "Look-back window in hours (default 24)",      "schema": { "type": "integer", "default": 24, "minimum": 1 } },
          { "name": "from",      "in": "query", "description": "Start of time range (RFC3339). Overrides hours.", "schema": { "type": "string", "format": "date-time" } },
          { "name": "to",        "in": "query", "description": "End of time range (RFC3339). Defaults to now.", "schema": { "type": "string", "format": "date-time" } },
          { "name": "scheduled", "in": "query", "description": "When true, returns only scheduled runs",       "schema": { "type": "boolean" } }
        ],
        "responses": {
          "200": { "description": "Paginated run summaries", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/PagedHistory" } } } },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/history/{id}": {
      "get": {
        "summary": "Get a test run by ID",
        "description": "Returns the full result of a specific test run including all DNS query and ping results.",
        "operationId": "getHistoryById",
        "tags": ["History"],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "description": "Run ID", "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Full test run", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/TestRun" } } } },
          "404": { "description": "Not found", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/compare": {
      "get": {
        "summary": "Compare two test runs",
        "description": "Computes per-server response-time deltas between two historical runs.",
        "operationId": "compareRuns",
        "tags": ["History"],
        "parameters": [
          { "name": "a", "in": "query", "required": true, "description": "ID of run A (baseline)",    "schema": { "type": "string" } },
          { "name": "b", "in": "query", "required": true, "description": "ID of run B (comparison)",  "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Comparison result", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/CompareResult" } } } },
          "400": { "description": "Missing query params", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "404": { "description": "Run not found",        "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },

    "/trends": {
      "get": {
        "summary": "Response-time trends over time",
        "description": "Returns per-server average DNS response times aggregated into time buckets. Uses hourly buckets for windows ≤ 48 h, daily buckets otherwise. Only successful (status=ok) queries are included.",
        "operationId": "getTrends",
        "tags": ["Trends"],
        "parameters": [
          { "name": "hours", "in": "query", "description": "Look-back window in hours (default 168 = 7 days, max 8760 = 1 year)", "schema": { "type": "integer", "default": 168, "minimum": 1, "maximum": 8760 } }
        ],
        "responses": {
          "200": { "description": "List of trend data points", "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/TrendPoint" } } } } },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },

    "/settings": {
      "get": {
        "summary": "Get current settings",
        "description": "Returns the current configuration including DNS servers, FQDNs, scheduled scans, and general options.",
        "operationId": "getSettings",
        "tags": ["Settings"],
        "responses": {
          "200": { "description": "Current settings", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Settings" } } } },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      },
      "put": {
        "summary": "Replace settings",
        "description": "Replaces the full settings object atomically. Fetch first, mutate, then PUT.",
        "operationId": "updateSettings",
        "tags": ["Settings"],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Settings" } } }
        },
        "responses": {
          "200": { "description": "Updated settings", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Settings" } } } },
          "400": { "description": "Invalid request", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/config/backup": {
      "post": {
        "summary": "Back up current config",
        "description": "Saves the current configuration to a backup file on disk (overwrites any previous backup).",
        "operationId": "backupConfig",
        "tags": ["Settings"],
        "responses": {
          "204": { "description": "Backup created" },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/config/restore": {
      "post": {
        "summary": "Restore config from backup",
        "description": "Overwrites the current configuration with the most recent backup and returns the restored settings.",
        "operationId": "restoreConfig",
        "tags": ["Settings"],
        "responses": {
          "200": { "description": "Restored settings", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Settings" } } } },
          "404": { "description": "No backup found", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/config/export": {
      "get": {
        "summary": "Export config as JSON",
        "description": "Returns the current configuration as a downloadable JSON file.",
        "operationId": "exportConfig",
        "tags": ["Settings"],
        "responses": {
          "200": {
            "description": "Config JSON file",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Settings" } } },
            "headers": {
              "Content-Disposition": { "schema": { "type": "string" }, "description": "attachment; filename=\"dnstester-config.json\"" }
            }
          },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/config/import": {
      "post": {
        "summary": "Import config from JSON",
        "description": "Replaces the current configuration with the uploaded JSON body and returns the applied settings.",
        "operationId": "importConfig",
        "tags": ["Settings"],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Settings" } } }
        },
        "responses": {
          "200": { "description": "Applied settings", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Settings" } } } },
          "400": { "description": "Invalid config", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },

    "/schedules": {
      "get": {
        "summary": "List scheduled scans",
        "description": "Returns all configured scheduled scan definitions.",
        "operationId": "listSchedules",
        "tags": ["Settings"],
        "responses": {
          "200": { "description": "List of schedules", "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/ScheduledScan" } } } } },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      },
      "post": {
        "summary": "Create a scheduled scan",
        "description": "Adds a new scheduled scan. The id field is ignored and generated server-side.",
        "operationId": "createSchedule",
        "tags": ["Settings"],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ScheduledScan" } } }
        },
        "responses": {
          "201": { "description": "Schedule created", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ScheduledScan" } } } },
          "400": { "description": "Invalid request", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/schedules/{id}": {
      "put": {
        "summary": "Update a scheduled scan",
        "description": "Replaces the definition of an existing scheduled scan.",
        "operationId": "updateSchedule",
        "tags": ["Settings"],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "description": "Schedule ID", "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ScheduledScan" } } }
        },
        "responses": {
          "200": { "description": "Updated schedule", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ScheduledScan" } } } },
          "400": { "description": "Invalid request", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "404": { "description": "Not found",       "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Internal error",  "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      },
      "delete": {
        "summary": "Delete a scheduled scan",
        "description": "Removes a scheduled scan by ID.",
        "operationId": "deleteSchedule",
        "tags": ["Settings"],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "description": "Schedule ID", "schema": { "type": "string" } }
        ],
        "responses": {
          "204": { "description": "Deleted" },
          "404": { "description": "Not found",      "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Internal error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },

    "/version": {
      "get": {
        "summary": "Get current binary version",
        "description": "Returns the version string injected at build time via ldflags.",
        "operationId": "getVersion",
        "tags": ["Updates"],
        "responses": {
          "200": { "description": "Version info", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/VersionInfo" } } } }
        }
      }
    },
    "/update/check": {
      "get": {
        "summary": "Check for a newer release",
        "description": "Queries the GitHub releases API and returns the latest version alongside the current one. The available field is false when running in dev mode or already on the latest version.",
        "operationId": "checkUpdate",
        "tags": ["Updates"],
        "responses": {
          "200": { "description": "Update check result", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/UpdateInfo" } } } },
          "502": { "description": "Could not reach GitHub API", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/update/apply": {
      "post": {
        "summary": "Download and apply an update",
        "description": "Downloads the binary at download_url, atomically replaces the running executable, responds with status=restarting, then calls os.Exit(0) so the process manager restarts with the new binary.",
        "operationId": "applyUpdate",
        "tags": ["Updates"],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ApplyUpdateRequest" } } }
        },
        "responses": {
          "200": { "description": "Update applied — server is restarting", "content": { "application/json": { "schema": { "type": "object", "properties": { "status": { "type": "string", "example": "restarting" } } } } } },
          "400": { "description": "download_url missing",   "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Download or replace failed", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    }
  },

  "components": {
    "schemas": {
      "DNSServer": {
        "type": "object",
        "required": ["name", "address", "enabled"],
        "properties": {
          "name":     { "type": "string", "example": "Cloudflare" },
          "address":  { "type": "string", "example": "1.1.1.1", "description": "IP, hostname, or full URL for DoH servers" },
          "protocol": { "type": "string", "enum": ["udp", "dot", "doh"], "description": "Transport protocol. Omitted or 'udp' = plain DNS/53, 'dot' = DNS-over-TLS/853, 'doh' = DNS-over-HTTPS" },
          "enabled":  { "type": "boolean" }
        }
      },
      "Settings": {
        "type": "object",
        "properties": {
          "servers":     { "type": "array",   "items": { "$ref": "#/components/schemas/DNSServer" } },
          "fqdns":       { "type": "array",   "items": { "type": "string" }, "example": ["google.com","github.com"] },
          "schedules":   { "type": "array",   "items": { "$ref": "#/components/schemas/ScheduledScan" } },
          "auto_update": { "type": "boolean", "description": "Enable periodic update checks (default false)" }
        }
      },
      "ScheduledScan": {
        "type": "object",
        "required": ["name", "enabled", "type"],
        "properties": {
          "id":               { "type": "string",  "readOnly": true },
          "name":             { "type": "string",  "example": "Hourly check" },
          "enabled":          { "type": "boolean" },
          "type":             { "type": "string",  "enum": ["interval","daily","weekdays","weekly","monthly","once"] },
          "interval_minutes": { "type": "integer", "description": "Required when type=interval", "example": 60 },
          "time_of_day":      { "type": "string",  "description": "HH:MM 24-hour. Required for daily/weekdays/weekly/monthly.", "example": "03:00" },
          "weekdays":         { "type": "array",   "items": { "type": "integer", "minimum": 0, "maximum": 6 }, "description": "0=Sun…6=Sat. Required for type=weekdays." },
          "weekday":          { "type": "integer", "minimum": 0, "maximum": 6, "description": "Required for type=weekly." },
          "day_of_month":     { "type": "integer", "minimum": 1, "maximum": 31, "description": "Required for type=monthly." },
          "run_at":           { "type": "string",  "format": "date-time", "description": "RFC3339. Required for type=once." }
        }
      },
      "QueryResult": {
        "type": "object",
        "properties": {
          "server_name": { "type": "string" },
          "server_addr": { "type": "string" },
          "protocol":    { "type": "string", "enum": ["udp","dot","doh"], "description": "Transport used for this query" },
          "fqdn":        { "type": "string" },
          "response_ms": { "type": "number", "format": "double" },
          "status":      { "type": "string", "enum": ["ok","error","timeout"] },
          "answers":     { "type": "array",  "items": { "type": "string" } },
          "error":       { "type": "string" },
          "timestamp":   { "type": "string", "format": "date-time" }
        }
      },
      "PingResult": {
        "type": "object",
        "properties": {
          "server_name": { "type": "string" },
          "server_addr": { "type": "string" },
          "latency_ms":  { "type": "number", "format": "double" },
          "status":      { "type": "string", "enum": ["ok","error","timeout"] },
          "error":       { "type": "string" }
        }
      },
      "TestRun": {
        "type": "object",
        "properties": {
          "id":           { "type": "string" },
          "started_at":   { "type": "string", "format": "date-time" },
          "completed_at": { "type": "string", "format": "date-time" },
          "status":       { "type": "string", "enum": ["running","completed"] },
          "schedule_id":  { "type": "string" },
          "dns_results":  { "type": "array", "items": { "$ref": "#/components/schemas/QueryResult" } },
          "ping_results": { "type": "array", "items": { "$ref": "#/components/schemas/PingResult" } }
        }
      },
      "RunSummary": {
        "type": "object",
        "properties": {
          "id":              { "type": "string" },
          "started_at":      { "type": "string", "format": "date-time" },
          "completed_at":    { "type": "string", "format": "date-time" },
          "status":          { "type": "string", "enum": ["running","completed"] },
          "schedule_id":     { "type": "string" },
          "total_queries":   { "type": "integer" },
          "success_count":   { "type": "integer" },
          "avg_response_ms": { "type": "number", "format": "double" }
        }
      },
      "PagedHistory": {
        "type": "object",
        "description": "Paginated response for GET /history",
        "properties": {
          "total": { "type": "integer", "description": "Total number of runs matching the filter" },
          "items": { "type": "array", "items": { "$ref": "#/components/schemas/RunSummary" } }
        }
      },
      "ServerStat": {
        "type": "object",
        "properties": {
          "server_name": { "type": "string" },
          "server_addr": { "type": "string" },
          "avg_ms_a":    { "type": "number", "format": "double" },
          "avg_ms_b":    { "type": "number", "format": "double" },
          "delta_ms":    { "type": "number", "format": "double" },
          "delta_pct":   { "type": "number", "format": "double" },
          "success_a":   { "type": "integer" },
          "success_b":   { "type": "integer" },
          "total_a":     { "type": "integer" },
          "total_b":     { "type": "integer" }
        }
      },
      "CompareResult": {
        "type": "object",
        "properties": {
          "run_a":             { "$ref": "#/components/schemas/TestRun" },
          "run_b":             { "$ref": "#/components/schemas/TestRun" },
          "by_server":         { "type": "array", "items": { "$ref": "#/components/schemas/ServerStat" } },
          "overall_delta_ms":  { "type": "number", "format": "double" },
          "overall_delta_pct": { "type": "number", "format": "double" }
        }
      },
      "TrendPoint": {
        "type": "object",
        "description": "One aggregated data point from GET /trends",
        "properties": {
          "server_name":  { "type": "string" },
          "server_addr":  { "type": "string" },
          "protocol":     { "type": "string", "description": "Transport protocol (omitted for UDP)" },
          "bucket":       { "type": "string", "description": "Time bucket: 'YYYY-MM-DD' (daily) or 'YYYY-MM-DD HH:00' (hourly)", "example": "2026-05-20" },
          "avg_ms":       { "type": "number", "format": "double", "description": "Average successful response time in milliseconds" },
          "sample_count": { "type": "integer", "description": "Number of successful queries in this bucket" }
        }
      },
      "VersionInfo": {
        "type": "object",
        "properties": {
          "version": { "type": "string", "example": "2026.5.5" }
        }
      },
      "UpdateInfo": {
        "type": "object",
        "properties": {
          "current":       { "type": "string", "description": "Running binary version" },
          "latest":        { "type": "string", "description": "Latest GitHub release tag" },
          "available":     { "type": "boolean", "description": "True when latest != current and current != 'dev'" },
          "release_notes": { "type": "string", "description": "Markdown body of the latest release" },
          "published_at":  { "type": "string", "format": "date-time" },
          "release_url":   { "type": "string", "format": "uri", "description": "GitHub release page URL" },
          "download_url":  { "type": "string", "format": "uri", "description": "Direct asset download URL for the current platform" }
        }
      },
      "ApplyUpdateRequest": {
        "type": "object",
        "required": ["download_url"],
        "properties": {
          "download_url": { "type": "string", "format": "uri", "description": "Direct binary download URL (from UpdateInfo.download_url)" }
        }
      }
    }
  }
}`
