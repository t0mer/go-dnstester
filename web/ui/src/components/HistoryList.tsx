import { api } from '../api/client'
import type { RunSummary, ScheduledScan, TestRun } from '../types'

interface Props {
  history: RunSummary[]
  schedules: ScheduledScan[]
  activeId?: string
  baselineId?: string
  onView: (run: TestRun) => void
  onSetBaseline: (run: TestRun | null) => void
}

function timeLabel(iso: string) {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'medium' })
}

export function HistoryList({ history, schedules, activeId, baselineId, onView, onSetBaseline }: Props) {
  const scheduleMap = Object.fromEntries(schedules.map(s => [s.id, s.name]))

  const load = async (summary: RunSummary, cb: (run: TestRun) => void) => {
    const run = await api.getRun(summary.id)
    cb(run)
  }

  if (history.length === 0) return <p className="text-gray-400 dark:text-gray-500 text-sm p-4">No past runs yet.</p>

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
          <tr>
            <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">Time</th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">Queries</th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">Success</th>
            <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">Avg (ms)</th>
            <th className="px-4 py-3" />
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
          {history.map(run => {
            const isActive   = run.id === activeId
            const isBaseline = run.id === baselineId
            const scheduleName = run.schedule_id ? scheduleMap[run.schedule_id] : null
            const pct = run.total_queries > 0
              ? Math.round((run.success_count / run.total_queries) * 100)
              : 0

            return (
              <tr key={run.id} className={`hover:bg-gray-50 dark:hover:bg-gray-800 ${isActive ? 'bg-blue-50 dark:bg-blue-950/50' : ''}`}>
                <td className="px-4 py-2.5 whitespace-nowrap">
                  <span className="text-gray-800 dark:text-gray-200">{timeLabel(run.started_at)}</span>
                  {scheduleName && (
                    <span className="ml-2 text-xs text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 px-1.5 py-0.5 rounded">
                      🕐 {scheduleName}
                    </span>
                  )}
                  {isActive    && <span className="ml-2 text-xs text-blue-600 dark:text-blue-400 font-medium">viewing</span>}
                  {isBaseline  && <span className="ml-2 text-xs text-purple-600 dark:text-purple-400 font-medium">baseline</span>}
                </td>
                <td className="px-4 py-2.5 tabular-nums text-gray-600 dark:text-gray-300">{run.total_queries}</td>
                <td className="px-4 py-2.5 tabular-nums">
                  <span className={pct === 100 ? 'text-green-600 dark:text-green-400' : pct >= 80 ? 'text-yellow-600 dark:text-yellow-400' : 'text-red-600 dark:text-red-400'}>
                    {run.success_count}/{run.total_queries} ({pct}%)
                  </span>
                </td>
                <td className="px-4 py-2.5 tabular-nums text-gray-600 dark:text-gray-300">
                  {run.avg_response_ms > 0 ? `${run.avg_response_ms.toFixed(0)} ms` : '—'}
                </td>
                <td className="px-4 py-2.5">
                  <div className="flex items-center gap-2 justify-end">
                    <button
                      onClick={() => load(run, onView)}
                      disabled={isActive}
                      className="text-xs px-2.5 py-1 rounded bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-40 disabled:cursor-default transition-colors"
                    >
                      View
                    </button>
                    {isBaseline ? (
                      <button onClick={() => onSetBaseline(null)} className="text-xs px-2.5 py-1 rounded border border-purple-300 dark:border-purple-700 text-purple-700 dark:text-purple-400 hover:bg-purple-50 dark:hover:bg-purple-950 transition-colors">
                        Clear
                      </button>
                    ) : (
                      <button onClick={() => load(run, r => onSetBaseline(r))} className="text-xs px-2.5 py-1 rounded border border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
                        Baseline
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
