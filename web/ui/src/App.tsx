import { useState, useEffect } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './api/client'
import { TestRunner } from './components/TestRunner'
import { DNSTable } from './components/DNSTable'
import { ResponseChart } from './components/ResponseChart'
import { PingResults } from './components/PingResults'
import { ServerConfig } from './components/ServerConfig'
import { FQDNConfig } from './components/FQDNConfig'
import { ScheduleConfig } from './components/ScheduleConfig'
import { HistoryList } from './components/HistoryList'
import { CompareView } from './components/CompareView'
import { GeneralSettings } from './components/GeneralSettings'
import { UpdateModal } from './components/UpdateModal'
import type { TestRun } from './types'

const SKIPPED_VERSION_KEY = 'dnstester_skipped_version'

type Tab = 'results' | 'compare' | 'history' | 'settings'
const VALID_TABS: Tab[] = ['results', 'compare', 'history', 'settings']

function tabFromHash(): Tab {
  const h = window.location.hash.slice(1) as Tab
  return VALID_TABS.includes(h) ? h : 'results'
}

export default function App() {
  const [tab, _setTab] = useState<Tab>(tabFromHash)
  const [activeRun, setActiveRun] = useState<TestRun | null>(null)
  const [baseline, setBaseline] = useState<TestRun | null>(null)
  const [skippedVersion, setSkippedVersion] = useState<string>(
    () => localStorage.getItem(SKIPPED_VERSION_KEY) ?? ''
  )
  const [updateModalOpen, setUpdateModalOpen] = useState(false)
  const qc = useQueryClient()

  const setTab = (t: Tab) => {
    _setTab(t)
    window.history.replaceState(null, '', `#${t}`)
  }

  // Keep tab in sync when the user navigates with browser back/forward.
  useEffect(() => {
    const onHashChange = () => _setTab(tabFromHash())
    window.addEventListener('hashchange', onHashChange)
    return () => window.removeEventListener('hashchange', onHashChange)
  }, [])

  // Auto-load the latest scan so Results is never blank on first visit.
  const { data: latestRun } = useQuery({
    queryKey: ['latest'],
    queryFn: api.getLatest,
    retry: false,
    staleTime: Infinity,
  })
  useEffect(() => {
    if (latestRun && !activeRun) setActiveRun(latestRun)
  }, [latestRun]) // eslint-disable-line react-hooks/exhaustive-deps

  const { data: config, isLoading: configLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  const { data: updateInfo } = useQuery({
    queryKey: ['update-check'],
    queryFn: api.checkUpdate,
    enabled: !!(config?.auto_update),
    staleTime: 4 * 60 * 60 * 1000,
    refetchInterval: 4 * 60 * 60 * 1000,
    retry: false,
  })

  // Auto-open modal when a new (unskipped) update is detected.
  useEffect(() => {
    if (updateInfo?.available && updateInfo.latest !== skippedVersion) {
      setUpdateModalOpen(true)
    }
  }, [updateInfo]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleSkipUpdate = () => {
    if (!updateInfo) return
    localStorage.setItem(SKIPPED_VERSION_KEY, updateInfo.latest)
    setSkippedVersion(updateInfo.latest)
    setUpdateModalOpen(false)
  }

  const showUpdateBadge = !!(updateInfo?.available && updateInfo.latest === skippedVersion)

  const { data: history = [] } = useQuery({
    queryKey: ['history'],
    queryFn: () => api.listHistory(100),
    refetchInterval: 15_000,
  })

  const handleResult = (run: TestRun) => {
    setActiveRun(run)
    qc.invalidateQueries({ queryKey: ['history'] })
    setTab('results')
  }

  const handleView = (run: TestRun) => {
    setActiveRun(run)
    setTab('results')
  }

  const TABS: { id: Tab; label: string }[] = [
    { id: 'results', label: 'Results' },
    { id: 'compare', label: 'Compare' },
    { id: 'history', label: 'History' },
    { id: 'settings', label: 'Settings' },
  ]

  return (
    <div className="min-h-screen bg-gray-50">
      {updateModalOpen && updateInfo?.available && (
        <UpdateModal
          info={updateInfo}
          onSkip={handleSkipUpdate}
          onClose={() => setUpdateModalOpen(false)}
        />
      )}

      <header className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-semibold text-gray-900">DNS Tester</h1>
            {showUpdateBadge && (
              <button
                onClick={() => setUpdateModalOpen(true)}
                title={`Update available: ${updateInfo!.latest}`}
                className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-amber-100 text-amber-700 text-xs font-medium hover:bg-amber-200 transition-colors"
              >
                ↑ {updateInfo!.latest}
              </button>
            )}
          </div>
          {activeRun?.completed_at && (
            <p className="text-xs text-gray-500 mt-0.5">
              Run {new Date(activeRun.started_at).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'medium' })}
              {baseline && (
                <span className="ml-2 text-purple-600">
                  · comparing with {new Date(baseline.started_at).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'medium' })}
                  <button onClick={() => setBaseline(null)} className="ml-1 hover:underline">(clear)</button>
                </span>
              )}
            </p>
          )}
        </div>
        <TestRunner onResult={handleResult} />
      </header>

      <nav className="bg-white border-b border-gray-200 px-6">
        <div className="flex gap-1">
          {TABS.map(t => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                tab === t.id
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      </nav>

      <main className="p-6 max-w-7xl mx-auto">
        {tab === 'results' && (
          <div className="space-y-6">
            {!activeRun ? (
              <div className="text-center py-20 text-gray-400">
                <p className="text-lg">No results yet</p>
                <p className="text-sm mt-1">Click "Run Test" to start, or pick a run from History</p>
              </div>
            ) : (
              <>
                <section>
                  <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">
                    Avg Response Time per Server
                  </h2>
                  <div className="bg-white rounded-lg border border-gray-200 p-4">
                    <ResponseChart results={activeRun.dns_results} baseline={baseline?.dns_results} />
                  </div>
                </section>
                <section>
                  <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">
                    DNS Query Results
                  </h2>
                  <div className="bg-white rounded-lg border border-gray-200">
                    <DNSTable results={activeRun.dns_results} baseline={baseline?.dns_results} />
                  </div>
                </section>
                <section>
                  <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">
                    Ping Results
                  </h2>
                  <PingResults results={activeRun.ping_results} />
                </section>
              </>
            )}
          </div>
        )}

        {tab === 'compare' && <CompareView history={history} schedules={config?.schedules ?? []} />}

        {tab === 'history' && (
          <div className="bg-white rounded-lg border border-gray-200">
            <div className="px-5 py-4 border-b border-gray-200">
              <h2 className="text-base font-semibold text-gray-900">Test History</h2>
              <p className="text-xs text-gray-500 mt-0.5">
                <strong>View</strong> — load into Results · <strong>Baseline</strong> — diff against active run · 🕐 = scheduled run
              </p>
            </div>
            <HistoryList
              history={history}
              schedules={config?.schedules ?? []}
              activeId={activeRun?.id}
              baselineId={baseline?.id}
              onView={handleView}
              onSetBaseline={setBaseline}
            />
          </div>
        )}

        {tab === 'settings' && (
          <div className="space-y-6">
            {configLoading ? (
              <p className="text-gray-500">Loading config…</p>
            ) : config ? (
              <>
                <GeneralSettings config={config} />
                <ScheduleConfig config={config} history={history} />
                <ServerConfig config={config} />
                <FQDNConfig config={config} />
              </>
            ) : null}
          </div>
        )}
      </main>
    </div>
  )
}
