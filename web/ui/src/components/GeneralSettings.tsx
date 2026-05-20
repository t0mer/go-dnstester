import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import type { Config } from '../types'

export function GeneralSettings({ config }: { config: Config }) {
  const qc = useQueryClient()
  const [autoUpdate, setAutoUpdate] = useState(config.auto_update ?? false)

  const save = useMutation({
    mutationFn: () => api.updateConfig({ ...config, auto_update: autoUpdate }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['config'] }),
  })

  return (
    <div className="bg-white border border-gray-200 rounded-lg">
      <div className="px-5 py-4 border-b border-gray-200">
        <h2 className="text-base font-semibold text-gray-900">General</h2>
      </div>
      <div className="px-5 py-4 space-y-4">
        <label className="flex items-start gap-3 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={autoUpdate}
            onChange={e => setAutoUpdate(e.target.checked)}
            className="mt-0.5 h-4 w-4 text-blue-600 rounded border-gray-300 cursor-pointer"
          />
          <div>
            <span className="text-sm font-medium text-gray-900">Enable update checks</span>
            <p className="text-xs text-gray-500 mt-0.5">
              Periodically check for new releases and show a notification when one is available.
            </p>
          </div>
        </label>
        <div className="flex justify-end">
          <button
            onClick={() => save.mutate()}
            disabled={save.isPending}
            className="btn-primary"
          >
            {save.isPending ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}
