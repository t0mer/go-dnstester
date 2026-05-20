import { useState, useEffect } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { api } from '../api/client'
import type { UpdateInfo } from '../types'

type UpdatePhase = 'idle' | 'downloading' | 'restarting'

interface Props {
  info: UpdateInfo
  onSkip: () => void
  onClose: () => void
}

export function UpdateModal({ info, onSkip, onClose }: Props) {
  const [phase, setPhase] = useState<UpdatePhase>('idle')
  const [error, setError] = useState<string | null>(null)

  // Poll for the server to come back up after restart, then reload.
  useEffect(() => {
    if (phase !== 'restarting') return
    const id = setInterval(async () => {
      try {
        await api.getVersion()
        clearInterval(id)
        window.location.reload()
      } catch {
        // server not up yet — keep polling
      }
    }, 2000)
    return () => clearInterval(id)
  }, [phase])

  const handleUpdate = async () => {
    if (!info.download_url) return
    setError(null)
    setPhase('downloading')
    try {
      await api.applyUpdate(info.download_url)
      setPhase('restarting')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Update failed')
      setPhase('idle')
    }
  }

  const isUpdating = phase !== 'idle'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-lg">

        <div className="px-6 py-5 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">Update Available</h2>
          <p className="text-sm text-gray-500 mt-0.5">A new version of DNS Tester is ready.</p>
        </div>

        <div className="px-6 py-4 space-y-4">
          {/* Version bar */}
          <div className="flex items-center gap-6 text-sm">
            <div>
              <p className="text-xs text-gray-400 uppercase tracking-wide mb-0.5">Current</p>
              <p className="font-mono font-medium text-gray-700">{info.current}</p>
            </div>
            <span className="text-gray-300 text-xl">→</span>
            <div>
              <p className="text-xs text-gray-400 uppercase tracking-wide mb-0.5">Latest</p>
              <p className="font-mono font-medium text-green-600">{info.latest}</p>
            </div>
            {info.published_at && (
              <div className="ml-auto text-right">
                <p className="text-xs text-gray-400 uppercase tracking-wide mb-0.5">Released</p>
                <p className="text-xs text-gray-500">
                  {new Date(info.published_at).toLocaleDateString(undefined, { dateStyle: 'medium' })}
                </p>
              </div>
            )}
          </div>

          {/* Changelog */}
          {info.release_notes && (
            <div>
              <p className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1">
                What's new
              </p>
              <div className="bg-gray-50 rounded-lg p-3 max-h-56 overflow-y-auto text-sm text-gray-700 leading-relaxed
                             prose prose-sm prose-gray max-w-none
                             prose-headings:text-sm prose-headings:font-semibold prose-headings:mt-2 prose-headings:mb-1
                             prose-a:text-blue-600 prose-a:no-underline hover:prose-a:underline
                             prose-li:my-0 prose-ul:my-1 prose-p:my-1">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>
                  {info.release_notes}
                </ReactMarkdown>
              </div>
            </div>
          )}

          {/* Restart status */}
          {phase === 'restarting' && (
            <div className="flex items-center gap-2 text-sm text-amber-700 bg-amber-50 rounded-lg px-3 py-2">
              <svg className="animate-spin h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"/>
              </svg>
              Restarting server — page will reload automatically…
            </div>
          )}

          {/* Error */}
          {error && (
            <p className="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>
          )}
        </div>

        <div className="px-6 py-4 border-t border-gray-100 flex items-center justify-between">
          <button
            onClick={onSkip}
            disabled={isUpdating}
            className="text-sm text-gray-400 hover:text-gray-600 transition-colors underline underline-offset-2 disabled:opacity-40"
          >
            Skip this version
          </button>
          <div className="flex items-center gap-2">
            <button onClick={onClose} disabled={isUpdating} className="btn-secondary disabled:opacity-40">
              Remind me later
            </button>
            <a
              href={info.release_url}
              target="_blank"
              rel="noopener noreferrer"
              className="btn-secondary"
            >
              View release ↗
            </a>
            {info.download_url && (
              <button
                onClick={handleUpdate}
                disabled={isUpdating}
                className="btn-primary disabled:opacity-60 min-w-[90px]"
              >
                {phase === 'idle' && 'Update'}
                {phase === 'downloading' && 'Updating…'}
                {phase === 'restarting' && 'Restarting…'}
              </button>
            )}
          </div>
        </div>

      </div>
    </div>
  )
}
