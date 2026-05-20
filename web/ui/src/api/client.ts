import type { AuthStatus, Config, TestRun, CompareResult, UpdateInfo, VersionInfo, PagedHistory, TrendPoint } from '../types'

const BASE = '/api'
export const TOKEN_STORAGE_KEY = 'dnst_api_token'

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  const storedToken = localStorage.getItem(TOKEN_STORAGE_KEY)
  if (storedToken) headers['Authorization'] = `Bearer ${storedToken}`

  const res = await fetch(`${BASE}${path}`, {
    method,
    credentials: 'same-origin',
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (res.status === 401) {
    window.dispatchEvent(new Event('auth:unauthorized'))
    throw new Error('Unauthorized')
  }
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(text || `${method} ${path} failed: ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  // Auth
  getAuthStatus: (): Promise<AuthStatus> => request('GET', '/auth/status'),
  login: (username: string, password: string): Promise<{ ok: boolean }> =>
    request('POST', '/auth/login', { username, password }),
  logout: (): Promise<void> => request('POST', '/auth/logout'),
  updateAuthSettings: (body: {
    enabled: boolean
    username: string
    password: string
    api_token_enabled: boolean
  }): Promise<AuthStatus> => request('PUT', '/auth/settings', body),
  generateToken: (): Promise<{ token: string }> => request('POST', '/auth/token'),
  revokeToken: (): Promise<void> => request('DELETE', '/auth/token'),

  // Config
  getConfig: (): Promise<Config> => request('GET', '/config'),
  updateConfig: (cfg: Config): Promise<Config> => request('PUT', '/config', cfg),
  backup: (): Promise<void> => request('POST', '/config/backup'),
  restore: (): Promise<Config> => request('POST', '/config/restore'),
  exportConfig: (): Promise<Blob> => fetch(`${BASE}/config/export`, { credentials: 'same-origin' }).then(r => r.blob()),
  importConfig: (data: string): Promise<Config> => {
    const parsed = JSON.parse(data) as Config
    return request('POST', '/config/import', parsed)
  },

  // Tests
  runTest: (): Promise<TestRun> => request('POST', '/test/run'),
  getLatest: (): Promise<TestRun> => request('GET', '/test/latest'),

  // History
  listHistory: (limit = 50, offset = 0, scheduledOnly = false): Promise<PagedHistory> =>
    request('GET', `/history?limit=${limit}&offset=${offset}${scheduledOnly ? '&scheduled=true' : ''}`),
  getRun: (id: string): Promise<TestRun> => request('GET', `/history/${id}`),
  compare: (idA: string, idB: string): Promise<CompareResult> =>
    request('GET', `/compare?a=${idA}&b=${idB}`),
  getTrends: (hours = 168): Promise<TrendPoint[]> =>
    request('GET', `/trends?hours=${hours}`),

  // Version / updates
  getVersion: (): Promise<VersionInfo> => request('GET', '/version'),
  checkUpdate: (): Promise<UpdateInfo> => request('GET', '/update/check'),
  applyUpdate: (downloadURL: string): Promise<{ status: string }> =>
    request('POST', '/update/apply', { download_url: downloadURL }),
}
