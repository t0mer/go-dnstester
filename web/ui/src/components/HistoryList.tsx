import { useState } from 'react'
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { api } from '../api/client'
import type { RunSummary, ScheduledScan, TestRun } from '../types'

interface Props {
  schedules: ScheduledScan[]
  activeId?: string
  baselineId?: string
  onView: (run: TestRun) => void
  onSetBaseline: (run: TestRun | null) => void
}

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100]

function timeLabel(iso: string) {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'medium' })
}

function pageNumbers(current: number, total: number): (number | '…')[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i)
  const pages: (number | '…')[] = []
  const add = (n: number | '…') => { if (pages[pages.length - 1] !== n) pages.push(n) }
  add(0)
  if (current > 2) add('…')
  for (let i = Math.max(1, current - 1); i <= Math.min(total - 2, current + 1); i++) add(i)
  if (current < total - 3) add('…')
  add(total - 1)
  return pages
}

export function HistoryList({ schedules, activeId, baselineId, onView, onSetBaseline }: Props) {
  const [page, setPage] = useState(0)
  const [pageSize, setPageSize] = useState(25)

  const scheduleMap = Object.fromEntries(schedules.map(s => [s.id, s.name]))

  const { data, isFetching } = useQuery({
    queryKey: ['history-paged', page, pageSize],
    queryFn: () => api.listHistory(pageSize, page * pageSize),
    placeholderData: keepPreviousData,
    refetchInterval: 15_000,
  })

  const items: RunSummary[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const load = async (summary: RunSummary, cb: (run: TestRun) => void) => {
    const run = await api.getRun(summary.id)
    cb(run)
  }

  const handlePageSize = (n: number) => {
    setPageSize(n)
    setPage(0)
  }

  const pages = pageNumbers(page, totalPages)

  const from = total === 0 ? 0 : page * pageSize + 1
  const to = Math.min(page * pageSize + pageSize, total)

  return (
    <div>
      {/* Table */}
      <div className="overflow-x-auto">
        {items.length === 0 && !isFetching ? (
          <p className="text-gray-400 dark:text-gray-500 text-sm p-4">No past runs yet.</p>
        ) : (
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
              <tr>
                <th className="px-3 sm:px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">Time</th>
                <th className="px-3 sm:px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide hidden sm:table-cell">Queries</th>
                <th className="px-3 sm:px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">Success</th>
                <th className="px-3 sm:px-4 py-3 text-left text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide hidden sm:table-cell">Avg (ms)</th>
                <th className="px-3 sm:px-4 py-3" />
              </tr>
            </thead>
            <tbody className={`divide-y divide-gray-100 dark:divide-gray-700 transition-opacity ${isFetching ? 'opacity-50' : ''}`}>
              {items.map(run => {
                const isActive   = run.id === activeId
                const isBaseline = run.id === baselineId
                const scheduleName = run.schedule_id ? scheduleMap[run.schedule_id] : null
                const pct = run.total_queries > 0
                  ? Math.round((run.success_count / run.total_queries) * 100)
                  : 0

                return (
                  <tr key={run.id} className={`hover:bg-gray-50 dark:hover:bg-gray-800 ${isActive ? 'bg-blue-50 dark:bg-blue-950/50' : ''}`}>
                    <td className="px-3 sm:px-4 py-3 min-w-0">
                      <div className="text-sm text-gray-800 dark:text-gray-200 whitespace-nowrap">{timeLabel(run.started_at)}</div>
                      <div className="flex flex-wrap gap-1 mt-0.5">
                        {scheduleName && (
                          <span className="text-xs text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 px-1.5 py-0.5 rounded">
                            🕐 {scheduleName}
                          </span>
                        )}
                        {isActive    && <span className="text-xs text-blue-600 dark:text-blue-400 font-medium">viewing</span>}
                        {isBaseline  && <span className="text-xs text-purple-600 dark:text-purple-400 font-medium">baseline</span>}
                      </div>
                    </td>
                    <td className="px-3 sm:px-4 py-3 tabular-nums text-sm text-gray-600 dark:text-gray-300 hidden sm:table-cell">{run.total_queries}</td>
                    <td className="px-3 sm:px-4 py-3 tabular-nums text-sm">
                      <span className={pct === 100 ? 'text-green-600 dark:text-green-400' : pct >= 80 ? 'text-yellow-600 dark:text-yellow-400' : 'text-red-600 dark:text-red-400'}>
                        {run.success_count}/{run.total_queries}
                        <span className="hidden sm:inline"> ({pct}%)</span>
                      </span>
                    </td>
                    <td className="px-3 sm:px-4 py-3 tabular-nums text-sm text-gray-600 dark:text-gray-300 hidden sm:table-cell">
                      {run.avg_response_ms > 0 ? `${run.avg_response_ms.toFixed(0)} ms` : '—'}
                    </td>
                    <td className="px-3 sm:px-4 py-3">
                      <div className="flex items-center gap-1.5 justify-end">
                        <button
                          onClick={() => load(run, onView)}
                          disabled={isActive}
                          className="text-xs px-2.5 py-1.5 rounded bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-40 disabled:cursor-default transition-colors"
                        >
                          View
                        </button>
                        {isBaseline ? (
                          <button onClick={() => onSetBaseline(null)} className="text-xs px-2.5 py-1.5 rounded border border-purple-300 dark:border-purple-700 text-purple-700 dark:text-purple-400 hover:bg-purple-50 dark:hover:bg-purple-950 transition-colors">
                            Clear
                          </button>
                        ) : (
                          <button onClick={() => load(run, r => onSetBaseline(r))} className="text-xs px-2.5 py-1.5 rounded border border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
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
        )}
      </div>

      {/* Pagination bar */}
      <div className="px-4 py-3 border-t border-gray-100 dark:border-gray-700 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        {/* Left: count + page size */}
        <div className="flex items-center gap-3 flex-wrap">
          <span className="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">
            {total === 0 ? 'No results' : `${from}–${to} of ${total}`}
          </span>
          <label className="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
            Per page:
            <select
              value={pageSize}
              onChange={e => handlePageSize(Number(e.target.value))}
              className="input py-0.5 px-1.5 text-xs"
            >
              {PAGE_SIZE_OPTIONS.map(n => (
                <option key={n} value={n}>{n}</option>
              ))}
            </select>
          </label>
        </div>

        {/* Right: page buttons */}
        {totalPages > 1 && (
          <div className="flex items-center gap-1 flex-wrap">
            <button
              onClick={() => setPage(p => p - 1)}
              disabled={page === 0}
              className="px-2.5 py-1.5 text-xs rounded border border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-40 disabled:cursor-default transition-colors"
            >
              ‹ Prev
            </button>

            {pages.map((p, i) =>
              p === '…' ? (
                <span key={`ellipsis-${i}`} className="px-1 text-xs text-gray-400 dark:text-gray-500">…</span>
              ) : (
                <button
                  key={p}
                  onClick={() => setPage(p as number)}
                  className={`min-w-[30px] px-2 py-1.5 text-xs rounded border transition-colors ${
                    p === page
                      ? 'bg-blue-600 text-white border-blue-600'
                      : 'border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800'
                  }`}
                >
                  {(p as number) + 1}
                </button>
              )
            )}

            <button
              onClick={() => setPage(p => p + 1)}
              disabled={page >= totalPages - 1}
              className="px-2.5 py-1.5 text-xs rounded border border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-40 disabled:cursor-default transition-colors"
            >
              Next ›
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
