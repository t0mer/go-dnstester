import { useMutation } from '@tanstack/react-query'
import { api } from '../api/client'
import type { TestRun } from '../types'

interface Props {
  onResult: (run: TestRun) => void
}

export function TestRunner({ onResult }: Props) {
  const { mutate, isPending, error } = useMutation({
    mutationFn: api.runTest,
    onSuccess: onResult,
  })

  return (
    <div className="flex items-center gap-3">
      {error && (
        <span className="text-sm text-red-600">Test failed. Check server.</span>
      )}
      <button
        onClick={() => mutate()}
        disabled={isPending}
        className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
      >
        {isPending ? (
          <>
            <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
            </svg>
            Running…
          </>
        ) : (
          'Run Test'
        )}
      </button>
    </div>
  )
}
