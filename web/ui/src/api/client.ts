import type { Config, TestRun, RunSummary, CompareResult } from '../types'

const BASE = '/api'

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(text || `${method} ${path} failed: ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  getConfig: (): Promise<Config> => request('GET', '/config'),
  updateConfig: (cfg: Config): Promise<Config> => request('PUT', '/config', cfg),
  backup: (): Promise<void> => request('POST', '/config/backup'),
  restore: (): Promise<Config> => request('POST', '/config/restore'),
  exportConfig: (): Promise<Blob> => fetch(`${BASE}/config/export`).then(r => r.blob()),
  importConfig: (data: string): Promise<Config> => {
    const parsed = JSON.parse(data) as Config
    return request('POST', '/config/import', parsed)
  },
  runTest: (): Promise<TestRun> => request('POST', '/test/run'),
  getLatest: (): Promise<TestRun> => request('GET', '/test/latest'),
  listHistory: (limit = 50, scheduledOnly = false): Promise<RunSummary[]> =>
    request('GET', `/history?limit=${limit}${scheduledOnly ? '&scheduled=true' : ''}`),
  getRun: (id: string): Promise<TestRun> => request('GET', `/history/${id}`),
  compare: (idA: string, idB: string): Promise<CompareResult> =>
    request('GET', `/compare?a=${idA}&b=${idB}`),
}
