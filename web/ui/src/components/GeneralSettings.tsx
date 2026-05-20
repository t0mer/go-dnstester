import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import type { Config } from '../types'

interface Props {
  config: Config
  dark: boolean
  onToggleDark: (v: boolean) => void
}

export function GeneralSettings({ config, dark, onToggleDark }: Props) {
  const qc = useQueryClient()
  const [autoUpdate, setAutoUpdate] = useState(config.auto_update ?? false)

  const { data: versionInfo } = useQuery({
    queryKey: ['version'],
    queryFn: api.getVersion,
    staleTime: Infinity,
  })

  const save = useMutation({
    mutationFn: () => api.updateConfig({ ...config, auto_update: autoUpdate }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['config'] }),
  })

  const checkNow = useMutation({
    mutationFn: api.checkUpdate,
    onSuccess: (info) => {
      qc.setQueryData(['update-check'], info)
    },
  })

  const checkResult = checkNow.data

  return (
    <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg">
      <div className="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
        <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">General</h2>
      </div>

      <div className="px-5 py-4 space-y-5">

        {/* Dark mode toggle */}
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Dark mode</p>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Switch between light and dark theme.</p>
          </div>
          <button
            onClick={() => onToggleDark(!dark)}
            role="switch"
            aria-checked={dark}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 ${
              dark ? 'bg-blue-600' : 'bg-gray-200 dark:bg-gray-600'
            }`}
          >
            <span className="sr-only">Toggle dark mode</span>
            <span className={`inline-flex h-4 w-4 items-center justify-center rounded-full bg-white shadow transform transition-transform text-[9px] ${
              dark ? 'translate-x-6' : 'translate-x-1'
            }`}>
              {dark ? '🌙' : '☀️'}
            </span>
          </button>
        </div>

        {/* Version + update check */}
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Current version</p>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5 font-mono">
              {versionInfo?.version ?? '—'}
            </p>
          </div>
          <div className="flex items-center gap-3">
            {checkResult && (
              checkResult.available
                ? <span className="text-xs text-amber-600 dark:text-amber-400 font-medium">↑ {checkResult.latest} available</span>
                : <span className="text-xs text-green-600 dark:text-green-400 font-medium">✓ Up to date</span>
            )}
            <button
              onClick={() => checkNow.mutate()}
              disabled={checkNow.isPending}
              className="btn-secondary text-sm"
            >
              {checkNow.isPending ? 'Checking…' : 'Check for updates'}
            </button>
          </div>
        </div>

        {/* Auto-update toggle */}
        <label className="flex items-start gap-3 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={autoUpdate}
            onChange={e => setAutoUpdate(e.target.checked)}
            className="mt-0.5 h-4 w-4 text-blue-600 rounded border-gray-300 dark:border-gray-600 cursor-pointer"
          />
          <div>
            <span className="text-sm font-medium text-gray-900 dark:text-gray-100">Enable automatic update checks</span>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
              Periodically check for new releases and show a notification when one is available.
            </p>
          </div>
        </label>

        <div className="flex justify-end">
          <button onClick={() => save.mutate()} disabled={save.isPending} className="btn-primary">
            {save.isPending ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}
