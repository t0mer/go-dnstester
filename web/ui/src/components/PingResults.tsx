import type { PingResult } from '../types'

interface Props {
  results: PingResult[]
}

const STATUS_COLOR: Record<string, string> = {
  ok: 'text-green-600',
  error: 'text-red-600',
  timeout: 'text-yellow-600',
}

export function PingResults({ results }: Props) {
  if (results.length === 0) return null
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
      {results.map((r, i) => (
        <div key={i} className="bg-white border border-gray-200 rounded-lg p-4">
          <p className="text-sm font-medium text-gray-900 truncate">{r.server_name}</p>
          <p className="text-xs text-gray-400 mb-2">{r.server_addr}</p>
          {r.status === 'ok' ? (
            <p className="text-xl font-semibold text-gray-800">
              {r.latency_ms.toFixed(1)}
              <span className="text-sm font-normal text-gray-400 ml-1">ms</span>
            </p>
          ) : (
            <p className={`text-sm font-medium ${STATUS_COLOR[r.status] ?? 'text-gray-500'}`}>
              {r.error ?? r.status}
            </p>
          )}
        </div>
      ))}
    </div>
  )
}
