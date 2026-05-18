import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  BarChart, Bar, Cell, XAxis, YAxis, CartesianGrid,
  Tooltip, Legend, ResponsiveContainer,
} from 'recharts'
import { api } from '../api/client'
import type { RunSummary, CompareResult, ServerStat, ScheduledScan } from '../types'

const COLORS = [
  '#3b82f6', '#10b981', '#f59e0b', '#ef4444',
  '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16',
  '#f97316', '#6366f1',
]

function deltaStyle(d: number) {
  if (d > 10) return 'text-red-600 font-semibold'
  if (d < -10) return 'text-green-600 font-semibold'
  return 'text-gray-500'
}

function arrow(d: number) {
  return d > 10 ? '▲' : d < -10 ? '▼' : '≈'
}

interface SelectorProps {
  label: string
  value: string
  onChange: (id: string) => void
  history: RunSummary[]
  scheduleMap: Record<string, string>
  exclude?: string
}

function RunSelector({ label, value, onChange, history, scheduleMap, exclude }: SelectorProps) {
  return (
    <div className="flex flex-col gap-1">
      <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide">{label}</label>
      <select value={value} onChange={e => onChange(e.target.value)} className="input text-sm min-w-[240px]">
        <option value="">— select a run —</option>
        {history.filter(r => r.id !== exclude).map(r => (
          <option key={r.id} value={r.id}>
            {new Date(r.started_at).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'medium' })}
            {r.schedule_id && scheduleMap[r.schedule_id] ? ` 🕐 ${scheduleMap[r.schedule_id]}` : ''}
            {` · ${r.success_count}/${r.total_queries} ok`}
            {` · ${r.avg_response_ms.toFixed(0)} ms avg`}
          </option>
        ))}
      </select>
    </div>
  )
}

function ServerCard({ stat, color }: { stat: ServerStat; color: string }) {
  const d = stat.delta_ms
  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4">
      <div className="flex items-center gap-2 mb-3">
        <span className="w-3 h-3 rounded-full flex-shrink-0" style={{ backgroundColor: color }} />
        <div>
          <p className="text-sm font-semibold text-gray-900">{stat.server_name}</p>
          <p className="text-xs text-gray-400">{stat.server_addr}</p>
        </div>
      </div>
      <div className="grid grid-cols-3 gap-2 text-center">
        <div>
          <p className="text-xs text-gray-400 mb-0.5">Run A</p>
          <p className="text-lg font-bold text-blue-600">{stat.avg_ms_a.toFixed(1)}<span className="text-xs font-normal text-gray-400 ml-0.5">ms</span></p>
        </div>
        <div>
          <p className="text-xs text-gray-400 mb-0.5">Δ</p>
          <p className={`text-lg font-bold ${deltaStyle(d)}`}>
            {arrow(d)} {Math.abs(d).toFixed(1)}<span className="text-xs font-normal ml-0.5">ms</span>
          </p>
          <p className={`text-xs ${deltaStyle(d)}`}>{stat.delta_pct > 0 ? '+' : ''}{stat.delta_pct.toFixed(1)}%</p>
        </div>
        <div>
          <p className="text-xs text-gray-400 mb-0.5">Run B</p>
          <p className="text-lg font-bold text-purple-600">{stat.avg_ms_b.toFixed(1)}<span className="text-xs font-normal text-gray-400 ml-0.5">ms</span></p>
        </div>
      </div>
    </div>
  )
}

function CompareChart({ result }: { result: CompareResult }) {
  const { data, colorMap } = useMemo(() => {
    const colorMap: Record<string, string> = {}
    result.by_server.forEach((s, i) => { colorMap[s.server_name] = COLORS[i % COLORS.length] })
    const data = [...result.by_server].sort((a, b) => a.avg_ms_a - b.avg_ms_a)
    return { data, colorMap }
  }, [result])

  return (
    <ResponsiveContainer width="100%" height={260}>
      <BarChart data={data} margin={{ top: 4, right: 16, left: 0, bottom: 4 }}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f0f0f0" />
        <XAxis dataKey="server_name" tick={{ fontSize: 12 }} />
        <YAxis unit="ms" tick={{ fontSize: 12 }} width={52} />
        <Tooltip formatter={(v: number, key: string) => [`${(v as number).toFixed(1)} ms`, key === 'avg_ms_a' ? 'Run A' : 'Run B']} contentStyle={{ fontSize: 12 }} />
        <Legend formatter={v => v === 'avg_ms_a' ? 'Run A' : 'Run B'} />
        <Bar dataKey="avg_ms_a" name="avg_ms_a" radius={[4, 4, 0, 0]}>
          {data.map((e, i) => <Cell key={i} fill={colorMap[e.server_name]} />)}
        </Bar>
        <Bar dataKey="avg_ms_b" name="avg_ms_b" radius={[4, 4, 0, 0]}>
          {data.map((e, i) => <Cell key={i} fill={colorMap[e.server_name]} fillOpacity={0.4} />)}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  )
}

interface Props {
  history: RunSummary[]
  schedules: ScheduledScan[]
}

export function CompareView({ history, schedules }: Props) {
  const [idA, setIdA] = useState('')
  const [idB, setIdB] = useState('')

  const scheduleMap = useMemo(
    () => Object.fromEntries(schedules.map(s => [s.id, s.name])),
    [schedules],
  )

  const { data: result, isFetching, error } = useQuery({
    queryKey: ['compare', idA, idB],
    queryFn: () => api.compare(idA, idB),
    enabled: Boolean(idA && idB),
  })

  const colorMap = useMemo(() => {
    if (!result) return {} as Record<string, string>
    const m: Record<string, string> = {}
    result.by_server.forEach((s, i) => { m[s.server_name] = COLORS[i % COLORS.length] })
    return m
  }, [result])

  // Quick-load last two runs for a specific schedule
  const loadScheduleRuns = (scheduleId: string) => {
    const runs = history.filter(r => r.schedule_id === scheduleId)
    if (runs.length >= 2) { setIdA(runs[1].id); setIdB(runs[0].id) }
  }

  // Schedules that have at least 2 runs available
  const schedulesWithRuns = schedules.filter(sc =>
    history.filter(r => r.schedule_id === sc.id).length >= 2
  )

  return (
    <div className="space-y-6">
      <div className="bg-white border border-gray-200 rounded-lg p-5">
        <div className="flex flex-wrap items-end gap-4">
          <RunSelector label="Run A (baseline)" value={idA} onChange={setIdA} history={history} scheduleMap={scheduleMap} exclude={idB} />
          <div className="text-gray-300 text-2xl pb-1">vs</div>
          <RunSelector label="Run B" value={idB} onChange={setIdB} history={history} scheduleMap={scheduleMap} exclude={idA} />
        </div>

        {schedulesWithRuns.length > 0 && (
          <div className="mt-3 flex items-center gap-2 flex-wrap">
            <span className="text-xs text-gray-500">Quick compare last two runs of:</span>
            {schedulesWithRuns.map(sc => (
              <button key={sc.id} onClick={() => loadScheduleRuns(sc.id)} className="btn-secondary text-xs px-2 py-1">
                🕐 {sc.name}
              </button>
            ))}
          </div>
        )}
      </div>

      {isFetching && <p className="text-gray-400 text-sm">Loading comparison…</p>}
      {error && <p className="text-red-600 text-sm">Failed to load comparison.</p>}

      {result && (
        <>
          <div className={`rounded-lg p-4 flex items-center gap-4 ${result.overall_delta_ms > 10 ? 'bg-red-50 border border-red-200' : result.overall_delta_ms < -10 ? 'bg-green-50 border border-green-200' : 'bg-gray-50 border border-gray-200'}`}>
            <span className="text-3xl">{result.overall_delta_ms > 10 ? '📈' : result.overall_delta_ms < -10 ? '📉' : '📊'}</span>
            <div>
              <p className="font-semibold text-gray-900">
                Overall: Run B is{' '}
                <span className={result.overall_delta_ms > 10 ? 'text-red-600' : result.overall_delta_ms < -10 ? 'text-green-600' : 'text-gray-700'}>
                  {Math.abs(result.overall_delta_ms).toFixed(1)} ms ({Math.abs(result.overall_delta_pct).toFixed(1)}%)
                  {' '}{result.overall_delta_ms > 10 ? 'slower' : result.overall_delta_ms < -10 ? 'faster' : 'similar'}
                </span>
                {' '}than Run A
              </p>
              <p className="text-xs text-gray-500 mt-0.5">
                A: {new Date(result.run_a.started_at).toLocaleString()} · B: {new Date(result.run_b.started_at).toLocaleString()}
              </p>
            </div>
          </div>

          <div className="bg-white border border-gray-200 rounded-lg p-4">
            <h3 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">
              Avg Response Time — A (solid) vs B (faded)
            </h3>
            <CompareChart result={result} />
          </div>

          <div>
            <h3 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Per-server breakdown</h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
              {[...result.by_server]
                .sort((a, b) => Math.abs(b.delta_ms) - Math.abs(a.delta_ms))
                .map((stat, i) => (
                  <ServerCard key={stat.server_name} stat={stat} color={colorMap[stat.server_name] ?? COLORS[i % COLORS.length]} />
                ))}
            </div>
          </div>
        </>
      )}

      {!idA && !idB && (
        <div className="text-center py-16 text-gray-400">
          <p className="text-lg">Select two runs above to compare them</p>
          {schedulesWithRuns.length > 0 && (
            <p className="text-sm mt-1">Or use a quick-compare button to load the last two runs of a schedule</p>
          )}
        </div>
      )}
    </div>
  )
}
