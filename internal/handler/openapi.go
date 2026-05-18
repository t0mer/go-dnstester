package handler

// openapiSpec is the OpenAPI 3.0 specification for the DNS Tester API.
const openapiSpec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "DNS Tester API",
    "description": "DNS performance testing: run ad-hoc or scheduled tests, browse history, and manage settings.",
    "version": "1.0.0"
  },
  "servers": [
    { "url": "/api", "description": "REST API" },
    { "url": "/",    "description": "Root (Prometheus metrics)" }
  ],
  "tags": [
    { "name": "Tests",     "description": "Run DNS tests and retrieve results" },
    { "name": "History",   "description": "Browse historical test runs" },
    { "name": "Settings",  "description": "Manage DNS servers, FQDNs, and global configuration" },
    { "name": "Schedules", "description": "Manage automated scheduled scans" },
    { "name": "Monitoring","description": "Prometheus metrics export" }
  ],
  "paths": {
    "/metrics": {
      "get": {
        "summary": "Prometheus metrics",
        "description": "Exposes metrics in Prometheus text format (served at root, not under /api). Scalar metrics (latest run): dnstester_dns_response_seconds{server_name,server_addr,fqdn,status}, dnstester_ping_latency_seconds{server_name,server_addr,status}, dnstester_last_run_timestamp_seconds, dnstester_last_run_duration_seconds, dnstester_test_runs_total. Per-run metrics (last 5 runs, labelled by run_id): dnstester_run_info{run_id,started_at,status}=1, dnstester_run_duration_seconds{run_id}, dnstester_run_dns_response_seconds{run_id,server_name,server_addr,fqdn,status}, dnstester_run_ping_latency_seconds{run_id,server_name,server_addr,status}. Plus standard Go runtime metrics.",
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
        "description": "Executes a DNS test against all enabled servers and FQDNs, persists the run to history, and returns the full result.",
        "operationId": "runTest",
        "tags": ["Tests"],
        "responses": {
          "200": {
            "description": "Test completed successfully",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/TestRun" } } }
          },
          "500": { "description": "Internal server error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      },
      "post": {
        "summary": "Run a DNS test (POST)",
        "description": "Identical to GET /test/run. Provided for clients that prefer POST for non-idempotent operations.",
        "operationId": "runTestPost",
        "tags": ["Tests"],
        "responses": {
          "200": {
            "description": "Test completed successfully",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/TestRun" } } }
          },
          "500": { "description": "Internal server error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/test/latest": {
      "get": {
        "summary": "Get the latest test run",
        "description": "Returns the most recent test run result (from memory if available, otherwise from the database).",
        "operationId": "getLatestTest",
        "tags": ["Tests"],
        "responses": {
          "200": {
            "description": "Latest test run",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/TestRun" } } }
          },
          "404": { "description": "No results yet", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/history": {
      "get": {
        "summary": "List test run history",
        "description": "Returns summaries of past test runs. Defaults to the last 24 hours. Provide explicit 'from'/'to' to override the time window.",
        "operationId": "listHistory",
        "tags": ["History"],
        "parameters": [
          {
            "name": "hours",
            "in": "query",
            "description": "Look-back window in hours (default 24). Ignored when 'from' or 'to' are provided.",
            "schema": { "type": "integer", "default": 24, "minimum": 1 }
          },
          {
            "name": "from",
            "in": "query",
            "description": "Start of time range (RFC3339, e.g. 2006-01-02T15:04:05Z). Overrides 'hours'.",
            "schema": { "type": "string", "format": "date-time" }
          },
          {
            "name": "to",
            "in": "query",
            "description": "End of time range (RFC3339). Defaults to now.",
            "schema": { "type": "string", "format": "date-time" }
          },
          {
            "name": "limit",
            "in": "query",
            "description": "Maximum number of results to return (default 100, max 500).",
            "schema": { "type": "integer", "default": 100, "minimum": 1, "maximum": 500 }
          },
          {
            "name": "scheduled",
            "in": "query",
            "description": "When true, returns only runs triggered by a schedule.",
            "schema": { "type": "boolean" }
          }
        ],
        "responses": {
          "200": {
            "description": "List of run summaries",
            "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/RunSummary" } } } }
          },
          "500": { "description": "Internal server error", "content": { "text/plain": { "schema": { "type": "string" } } } }
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
          {
            "name": "id",
            "in": "path",
            "required": true,
            "description": "Run ID (from a RunSummary)",
            "schema": { "type": "string" }
          }
        ],
        "responses": {
          "200": {
            "description": "Full test run",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/TestRun" } } }
          },
          "404": { "description": "Not found", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Internal server error", "content": { "text/plain": { "schema": { "type": "string" } } } }
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
          { "name": "a", "in": "query", "required": true, "description": "ID of run A (baseline)", "schema": { "type": "string" } },
          { "name": "b", "in": "query", "required": true, "description": "ID of run B (comparison)", "schema": { "type": "string" } }
        ],
        "responses": {
          "200": {
            "description": "Comparison result",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/CompareResult" } } }
          },
          "400": { "description": "Missing query params", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "404": { "description": "Run not found", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/settings": {
      "get": {
        "summary": "Get current settings",
        "description": "Returns the current configuration including DNS servers, FQDNs, and schedules.",
        "operationId": "getSettings",
        "tags": ["Settings"],
        "responses": {
          "200": {
            "description": "Current settings",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Settings" } } }
          },
          "500": { "description": "Internal server error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      },
      "put": {
        "summary": "Update settings",
        "description": "Replaces the full settings object. To add, modify, or remove servers/FQDNs/schedules, fetch the current settings first, mutate, then PUT the result.",
        "operationId": "updateSettings",
        "tags": ["Settings"],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Settings" } } }
        },
        "responses": {
          "200": {
            "description": "Updated settings",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Settings" } } }
          },
          "400": { "description": "Invalid request body", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Internal server error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/schedules": {
      "get": {
        "summary": "List scheduled scans",
        "description": "Returns all configured scheduled scan definitions.",
        "operationId": "listSchedules",
        "tags": ["Schedules"],
        "responses": {
          "200": {
            "description": "List of schedules",
            "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/ScheduledScan" } } } }
          },
          "500": { "description": "Internal server error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      },
      "post": {
        "summary": "Create a new scheduled scan",
        "description": "Adds a new scheduled scan. The 'id' field is ignored and generated server-side.",
        "operationId": "createSchedule",
        "tags": ["Schedules"],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ScheduledScan" } } }
        },
        "responses": {
          "201": {
            "description": "Schedule created",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ScheduledScan" } } }
          },
          "400": { "description": "Invalid request", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Internal server error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      }
    },
    "/schedules/{id}": {
      "put": {
        "summary": "Update an existing scheduled scan",
        "description": "Replaces the definition of an existing scheduled scan by ID.",
        "operationId": "updateSchedule",
        "tags": ["Schedules"],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "description": "Schedule ID", "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ScheduledScan" } } }
        },
        "responses": {
          "200": {
            "description": "Updated schedule",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ScheduledScan" } } }
          },
          "400": { "description": "Invalid request", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "404": { "description": "Schedule not found", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Internal server error", "content": { "text/plain": { "schema": { "type": "string" } } } }
        }
      },
      "delete": {
        "summary": "Delete a scheduled scan",
        "description": "Removes a scheduled scan by ID.",
        "operationId": "deleteSchedule",
        "tags": ["Schedules"],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "description": "Schedule ID", "schema": { "type": "string" } }
        ],
        "responses": {
          "204": { "description": "Deleted successfully" },
          "404": { "description": "Schedule not found", "content": { "text/plain": { "schema": { "type": "string" } } } },
          "500": { "description": "Internal server error", "content": { "text/plain": { "schema": { "type": "string" } } } }
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
          "name":    { "type": "string", "example": "Cloudflare" },
          "address": { "type": "string", "example": "1.1.1.1" },
          "enabled": { "type": "boolean" }
        }
      },
      "ScheduledScan": {
        "type": "object",
        "required": ["name", "enabled", "type"],
        "properties": {
          "id":               { "type": "string", "readOnly": true, "description": "Generated server-side on create" },
          "name":             { "type": "string", "example": "Hourly check" },
          "enabled":          { "type": "boolean" },
          "type":             { "type": "string", "enum": ["interval","daily","weekdays","weekly","monthly","once"] },
          "interval_minutes": { "type": "integer", "description": "Required when type=interval", "example": 60 },
          "time_of_day":      { "type": "string", "description": "HH:MM (24-hour). Required for daily/weekdays/weekly/monthly.", "example": "03:00" },
          "weekdays":         { "type": "array", "items": { "type": "integer", "minimum": 0, "maximum": 6 }, "description": "Days of week (0=Sun). Required for type=weekdays." },
          "weekday":          { "type": "integer", "minimum": 0, "maximum": 6, "description": "Day of week (0=Sun). Required for type=weekly." },
          "day_of_month":     { "type": "integer", "minimum": 1, "maximum": 31, "description": "Day within month. Required for type=monthly." },
          "run_at":           { "type": "string", "format": "date-time", "description": "Exact run time (RFC3339). Required for type=once." }
        }
      },
      "Settings": {
        "type": "object",
        "properties": {
          "servers":   { "type": "array", "items": { "$ref": "#/components/schemas/DNSServer" } },
          "fqdns":     { "type": "array", "items": { "type": "string" }, "example": ["google.com","github.com"] },
          "schedules": { "type": "array", "items": { "$ref": "#/components/schemas/ScheduledScan" } }
        }
      },
      "QueryResult": {
        "type": "object",
        "properties": {
          "server_name": { "type": "string" },
          "server_addr": { "type": "string" },
          "fqdn":        { "type": "string" },
          "response_ms": { "type": "number", "format": "double" },
          "status":      { "type": "string", "enum": ["ok","error","timeout"] },
          "answers":     { "type": "array", "items": { "type": "string" } },
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
          "run_a":              { "$ref": "#/components/schemas/TestRun" },
          "run_b":              { "$ref": "#/components/schemas/TestRun" },
          "by_server":          { "type": "array", "items": { "$ref": "#/components/schemas/ServerStat" } },
          "overall_delta_ms":   { "type": "number", "format": "double" },
          "overall_delta_pct":  { "type": "number", "format": "double" }
        }
      }
    }
  }
}`
