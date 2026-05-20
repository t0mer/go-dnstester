import type { UpdateInfo } from '../types'

interface Props {
  info: UpdateInfo
  onSkip: () => void
  onClose: () => void
}

export function UpdateModal({ info, onSkip, onClose }: Props) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-lg">
        <div className="px-6 py-5 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">Update Available</h2>
          <p className="text-sm text-gray-500 mt-0.5">
            A new version of DNS Tester is ready.
          </p>
        </div>

        <div className="px-6 py-4 space-y-4">
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

          {info.release_notes && (
            <div>
              <p className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1">
                What's new
              </p>
              <div className="bg-gray-50 rounded-lg p-3 max-h-48 overflow-y-auto text-xs text-gray-700 whitespace-pre-wrap leading-relaxed">
                {info.release_notes}
              </div>
            </div>
          )}
        </div>

        <div className="px-6 py-4 border-t border-gray-100 flex items-center justify-between">
          <button
            onClick={onSkip}
            className="text-sm text-gray-400 hover:text-gray-600 transition-colors underline underline-offset-2"
          >
            Skip this version
          </button>
          <div className="flex items-center gap-2">
            <button onClick={onClose} className="btn-secondary">
              Remind me later
            </button>
            <a
              href={info.release_url}
              target="_blank"
              rel="noopener noreferrer"
              className="btn-primary"
            >
              View release ↗
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}
