import { useState, useMemo } from 'react'
import type { QueryResult } from '../types'

type SortField = 'server_name' | 'fqdn' | 'response_ms' | 'status' | 'delta_ms'
type SortDir = 'asc' | 'desc'

interface Props {
  results: QueryResult[]
  baseline?: QueryResult[]
}

const STATUS_STYLE: Record<string, string> = {
  ok: 'bg-green-100 text-green-800',
  error: 'bg-red-100 text-red-800',
  timeout: 'bg-yellow-100 text-yellow-800',
}

function deltaClass(d: number): string {
  if (d > 20) return 'text-red-600 font-medium'
  if (d < -20) return 'text-green-600 font-medium'
  return 'text-gray-500'
}

export function DNSTable({ results, baseline }: Props) {
  const [filter, setFilter] = useState('')
  const [sortField, setSortField] = useState<SortField>('response_ms')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  const baselineMap = useMemo(() => {
    if (!baseline) return null
    const m = new Map<string, number>()
    for (const r of baseline) {
      if (r.status === 'ok') m.set(`${r.server_name}|${r.fqdn}`, r.response_ms)
    }
    return m
  }, [baseline])

  const rows = useMemo(() => {
    const q = filter.toLowerCase()
    const filtered = results.filter(r =>
      r.server_name.toLowerCase().includes(q) ||
      r.server_addr.toLowerCase().includes(q) ||
      r.fqdn.toLowerCase().includes(q) ||
      r.status.toLowerCase().includes(q),
    )
    return filtered
      .map(r => {
        const base = baselineMap?.get(`${r.server_name}|${r.fqdn}`)
        const delta = base !== undefined && r.status === 'ok' ? r.response_ms - base : null
        return { ...r, delta_ms: delta }
      })
      .sort((a, b) => {
        if (sortField === 'delta_ms') {
          const av = a.delta_ms ?? Infinity
          const bv = b.delta_ms ?? Infinity
          return sortDir === 'asc' ? av - bv : bv - av
        }
        const av = a[sortField as keyof QueryResult]
        const bv = b[sortField as keyof QueryResult]
        if (typeof av === 'number' && typeof bv === 'number')
          return sortDir === 'asc' ? av - bv : bv - av
        return sortDir === 'asc'
          ? String(av).localeCompare(String(bv))
          : String(bv).localeCompare(String(av))
      })
  }, [results, filter, sortField, sortDir, baselineMap])

  const col = (label: string, field: SortField, cls = '') => (
    <th
      onClick={() => {
        if (sortField === field) setSortDir(d => (d === 'asc' ? 'desc' : 'asc'))
        else { setSortField(field); setSortDir('asc') }
      }}
      className={`px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide cursor-pointer hover:bg-gray-50 select-none whitespace-nowrap ${cls}`}
    >
      {label}
      {sortField === field && <span className="ml-1">{sortDir === 'asc' ? '↑' : '↓'}</span>}
    </th>
  )

  return (
    <div>
      <div className="px-4 py-3 border-b border-gray-200">
        <input
          type="search"
          placeholder="Filter by server, FQDN, or status…"
          value={filter}
          onChange={e => setFilter(e.target.value)}
          className="w-full max-w-sm px-3 py-1.5 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              {col('Server', 'server_name')}
              {col('FQDN', 'fqdn')}
              {col('Response (ms)', 'response_ms')}
              {baselineMap && col('Δ vs baseline', 'delta_ms', 'text-purple-600')}
              {col('Status', 'status')}
              <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wide">Answer</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {rows.map((r, i) => (
              <tr key={i} className="hover:bg-gray-50">
                <td className="px-4 py-2.5 font-medium text-gray-900 whitespace-nowrap">
                  {r.server_name}
                  <span className="ml-1.5 text-xs text-gray-400">{r.server_addr}</span>
                </td>
                <td className="px-4 py-2.5 text-gray-600">{r.fqdn}</td>
                <td className="px-4 py-2.5 text-gray-700 tabular-nums">{r.response_ms.toFixed(1)}</td>
                {baselineMap && (
                  <td className={`px-4 py-2.5 tabular-nums ${r.delta_ms !== null ? deltaClass(r.delta_ms) : 'text-gray-300'}`}>
                    {r.delta_ms !== null
                      ? `${r.delta_ms > 0 ? '+' : ''}${r.delta_ms.toFixed(1)}`
                      : '—'}
                  </td>
                )}
                <td className="px-4 py-2.5">
                  <span className={`inline-flex px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_STYLE[r.status] ?? 'bg-gray-100 text-gray-600'}`}>
                    {r.status}
                  </span>
                </td>
                <td className="px-4 py-2.5 text-gray-500 text-xs">
                  {r.answers?.join(', ') ?? r.error ?? '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {rows.length === 0 && (
          <p className="text-center py-8 text-gray-400 text-sm">No results match the filter.</p>
        )}
      </div>
    </div>
  )
}
