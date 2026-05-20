import { useMemo } from 'react'
import {
  BarChart, Bar, Cell, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, Legend,
} from 'recharts'
import { useIsDark } from '../hooks/useDarkMode'
import type { QueryResult } from '../types'

interface Props {
  results: QueryResult[]
  baseline?: QueryResult[]
}

const COLORS = [
  '#3b82f6', '#10b981', '#f59e0b', '#ef4444',
  '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16',
  '#f97316', '#6366f1',
]

function avgByServer(results: QueryResult[]) {
  const m: Record<string, { total: number; count: number }> = {}
  for (const r of results) {
    if (r.status !== 'ok') continue
    if (!m[r.server_name]) m[r.server_name] = { total: 0, count: 0 }
    m[r.server_name].total += r.response_ms
    m[r.server_name].count++
  }
  return m
}

export function ResponseChart({ results, baseline }: Props) {
  const isDark = useIsDark()

  const { data, colorMap } = useMemo(() => {
    const current = avgByServer(results)
    const base = baseline ? avgByServer(baseline) : null

    const names = Array.from(new Set([...Object.keys(current), ...(base ? Object.keys(base) : [])]))
      .sort((a, b) => {
        const av = current[a] ? current[a].total / current[a].count : 9999
        const bv = current[b] ? current[b].total / current[b].count : 9999
        return av - bv
      })

    const colorMap: Record<string, string> = {}
    names.forEach((name, i) => { colorMap[name] = COLORS[i % COLORS.length] })

    const data = names.map(name => ({
      name,
      current: current[name] ? Math.round(current[name].total / current[name].count) : null,
      baseline: base && base[name] ? Math.round(base[name].total / base[name].count) : null,
    }))

    return { data, colorMap }
  }, [results, baseline])

  if (data.length === 0) return <p className="text-gray-400 dark:text-gray-500 text-sm py-4">No successful results to chart.</p>

  const tickColor = isDark ? '#9ca3af' : '#6b7280'
  const gridColor = isDark ? '#374151' : '#f0f0f0'
  const tooltipStyle = {
    fontSize: 12,
    backgroundColor: isDark ? '#1f2937' : '#fff',
    borderColor: isDark ? '#374151' : '#e5e7eb',
    color: isDark ? '#f3f4f6' : '#111827',
  }

  return (
    <ResponsiveContainer width="100%" height={240}>
      <BarChart data={data} margin={{ top: 4, right: 16, left: 0, bottom: 4 }}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={gridColor} />
        <XAxis dataKey="name" tick={{ fontSize: 12, fill: tickColor }} />
        <YAxis unit="ms" tick={{ fontSize: 12, fill: tickColor }} width={52} />
        <Tooltip
          formatter={(v: number, name: string) => [`${v} ms`, name === 'current' ? 'This run' : 'Baseline']}
          contentStyle={tooltipStyle}
        />
        {baseline && (
          <Legend formatter={v => v === 'current' ? 'This run' : 'Baseline'} />
        )}
        <Bar dataKey="current" radius={[4, 4, 0, 0]} name="current">
          {data.map((entry, i) => (
            <Cell key={i} fill={colorMap[entry.name]} />
          ))}
        </Bar>
        {baseline && (
          <Bar dataKey="baseline" radius={[4, 4, 0, 0]} name="baseline">
            {data.map((entry, i) => (
              <Cell key={i} fill={colorMap[entry.name]} fillOpacity={0.35} />
            ))}
          </Bar>
        )}
      </BarChart>
    </ResponsiveContainer>
  )
}
