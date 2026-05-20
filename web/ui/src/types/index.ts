export interface DNSServer {
  name: string
  address: string
  enabled: boolean
}

export type ScheduleType = 'interval' | 'daily' | 'weekdays' | 'weekly' | 'monthly' | 'once'

export interface ScheduledScan {
  id: string
  name: string
  enabled: boolean
  type: ScheduleType
  // interval
  interval_minutes?: number
  // daily / weekdays / weekly / monthly
  time_of_day?: string   // "HH:MM"
  // weekdays
  weekdays?: number[]    // 0=Sun … 6=Sat
  // weekly
  weekday?: number       // 0–6
  // monthly
  day_of_month?: number  // 1–31
  // once
  run_at?: string        // RFC3339
}

export interface Config {
  servers: DNSServer[]
  fqdns: string[]
  schedules: ScheduledScan[]
  auto_update: boolean
}

export interface UpdateInfo {
  current: string
  latest: string
  available: boolean
  release_notes: string
  published_at: string
  release_url: string
}

export interface QueryResult {
  server_name: string
  server_addr: string
  fqdn: string
  response_ms: number
  status: 'ok' | 'error' | 'timeout'
  answers?: string[]
  error?: string
  timestamp: string
}

export interface PingResult {
  server_name: string
  server_addr: string
  latency_ms: number
  status: 'ok' | 'error' | 'timeout'
  error?: string
}

export interface TestRun {
  id: string
  started_at: string
  completed_at?: string
  dns_results: QueryResult[]
  ping_results: PingResult[]
  status: 'running' | 'completed'
  schedule_id: string
}

export interface RunSummary {
  id: string
  started_at: string
  completed_at?: string
  status: 'running' | 'completed'
  schedule_id: string
  total_queries: number
  success_count: number
  avg_response_ms: number
}

export interface ServerStat {
  server_name: string
  server_addr: string
  avg_ms_a: number
  avg_ms_b: number
  delta_ms: number
  delta_pct: number
  success_a: number
  success_b: number
  total_a: number
  total_b: number
}

export interface CompareResult {
  run_a: TestRun
  run_b: TestRun
  by_server: ServerStat[]
  overall_delta_ms: number
  overall_delta_pct: number
}
