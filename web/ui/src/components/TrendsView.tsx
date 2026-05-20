import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, Legend, ResponsiveContainer,
} from 'recharts'
import { useIsDark } from '../hooks/useDarkMode'
import { api } from '../api/client'

const COLORS = [
  '#3b82f6', '#10b981', '#f59e0b', '#ef4444',
  '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16',
  '#f97316', '#6366f1',
]

const RANGES = [
  { label: '24 h',   hours: 24  },
  { label: '7 days', hours: 168 },
  { label: '30 days', hours: 720 },
]

const MONTHS = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec']

function formatBucket(bucket: string, hours: number): string {
  if (hours <= 48) {
    // "2026-05-20 10:00" → "May 20 10:00"
    const [date, time] = bucket.split(' ')
    const [, m, d] = date.split('-').map(Number)
    return `${MONTHS[m - 1]} ${d} ${time}`
  }
  // "2026-05-20" → "May 20"
  const [, m, d] = bucket.split('-').map(Number)
  return `${MONTHS[m - 1]} ${d}`
}

function serverKey(name: string, protocol?: string): string {
  if (protocol && protocol !== 'udp') return `${name} (${protocol.toUpperCase()})`
  return name
}

export function TrendsView() {
  const [hours, setHours] = useState(168)
  const isDark = useIsDark()

  const { data = [], isFetching } = useQuery({
    queryKey: ['trends', hours],
    queryFn: () => api.getTrends(hours),
    staleTime: 60_000,
    refetchInterval: 60_000,
  })

  const { chartData, seriesKeys, colorMap } = useMemo(() => {
    // Unique series keys (server name + optional protocol tag)
    const keyOrder: string[] = []
    const keySet = new Set<string>()
    for (const p of data) {
      const k = serverKey(p.server_name, p.protocol)
      if (!keySet.has(k)) { keySet.add(k); keyOrder.push(k) }
    }

    // Build colour map
    const colorMap: Record<string, string> = {}
    keyOrder.forEach((k, i) => { colorMap[k] = COLORS[i % COLORS.length] })

    // Pivot: bucket → { bucket, [seriesKey]: avg_ms }
    const bucketMap = new Map<string, Record<string, unknown>>()
    for (const p of data) {
      if (!bucketMap.has(p.bucket)) bucketMap.set(p.bucket, { bucket: p.bucket })
      bucketMap.get(p.bucket)![serverKey(p.server_name, p.protocol)] = p.avg_ms
    }

    const chartData = [...bucketMap.values()].sort((a, b) =>
      String(a.bucket).localeCompare(String(b.bucket))
    )

    return { chartData, seriesKeys: keyOrder, colorMap }
  }, [data])

  const tickColor = isDark ? '#9ca3af' : '#6b7280'
  const gridColor = isDark ? '#374151' : '#f0f0f0'
  const tooltipStyle = {
    fontSize: 12,
    backgroundColor: isDark ? '#1f2937' : '#fff',
    borderColor: isDark ? '#374151' : '#e5e7eb',
    color: isDark ? '#f3f4f6' : '#111827',
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h2 className="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">
            Response Time Trends
          </h2>
          <p className="text-xs text-gray-400 dark:text-gray-500 mt-0.5">
            Average successful query time per server — {hours <= 48 ? 'hourly' : 'daily'} buckets
          </p>
        </div>
        <div className="flex items-center gap-1">
          {RANGES.map(r => (
            <button
              key={r.hours}
              onClick={() => setHours(r.hours)}
              className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                hours === r.hours
                  ? 'bg-blue-600 text-white'
                  : 'bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-300 border border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700'
              }`}
            >
              {r.label}
            </button>
          ))}
        </div>
      </div>

      {/* Chart card */}
      <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
        {chartData.length === 0 ? (
          <div className={`flex items-center justify-center h-64 ${isFetching ? 'opacity-40' : ''}`}>
            <div className="text-center text-gray-400 dark:text-gray-500">
              <p className="text-base">{isFetching ? 'Loading…' : 'No data for this period'}</p>
              {!isFetching && (
                <p className="text-sm mt-1">Run some tests to start building trends</p>
              )}
            </div>
          </div>
        ) : (
          <div className={`transition-opacity ${isFetching ? 'opacity-50' : 'opacity-100'}`}>
            <ResponsiveContainer width="100%" height={380}>
              <LineChart data={chartData} margin={{ top: 4, right: 16, left: 0, bottom: 4 }}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={gridColor} />
                <XAxis
                  dataKey="bucket"
                  tick={{ fontSize: 11, fill: tickColor }}
                  tickFormatter={b => formatBucket(String(b), hours)}
                  interval="preserveStartEnd"
                  minTickGap={60}
                />
                <YAxis
                  unit="ms"
                  tick={{ fontSize: 11, fill: tickColor }}
                  width={52}
                  domain={[0, 'auto']}
                />
                <Tooltip
                  contentStyle={tooltipStyle}
                  labelFormatter={b => formatBucket(String(b), hours)}
                  formatter={(v: number, name: string) => [`${v.toFixed(1)} ms`, name]}
                />
                <Legend wrapperStyle={{ fontSize: 12 }} />
                {seriesKeys.map(key => (
                  <Line
                    key={key}
                    type="monotone"
                    dataKey={key}
                    stroke={colorMap[key]}
                    strokeWidth={2}
                    dot={false}
                    activeDot={{ r: 4, strokeWidth: 0 }}
                    connectNulls={false}
                  />
                ))}
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      {/* Summary table */}
      {chartData.length > 0 && (
        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
          <div className="px-5 py-3 border-b border-gray-100 dark:border-gray-700">
            <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">
              Period averages
            </h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th className="px-4 py-2.5 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">Server</th>
                  <th className="px-4 py-2.5 text-right text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">Avg (ms)</th>
                  <th className="px-4 py-2.5 text-right text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">Samples</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {seriesKeys.map((key, i) => {
                  const pts = data.filter(p => serverKey(p.server_name, p.protocol) === key)
                  const total = pts.reduce((s, p) => s + p.avg_ms * p.sample_count, 0)
                  const count = pts.reduce((s, p) => s + p.sample_count, 0)
                  const avg = count > 0 ? total / count : 0
                  return (
                    <tr key={key} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                      <td className="px-4 py-2.5 flex items-center gap-2">
                        <span
                          className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                          style={{ backgroundColor: COLORS[i % COLORS.length] }}
                        />
                        <span className="text-gray-900 dark:text-gray-100 font-medium">{key}</span>
                      </td>
                      <td className="px-4 py-2.5 text-right tabular-nums text-gray-700 dark:text-gray-300">
                        {avg.toFixed(1)}
                      </td>
                      <td className="px-4 py-2.5 text-right tabular-nums text-gray-500 dark:text-gray-400">
                        {count.toLocaleString()}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
