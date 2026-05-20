import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { TOKEN_STORAGE_KEY } from '../api/client'
import { api } from '../api/client'

interface Props {
  onSuccess: () => void
}

export function TokenGatePage({ onSuccess }: Props) {
  const [token, setToken] = useState('')
  const [error, setError] = useState('')
  const [checking, setChecking] = useState(false)
  const qc = useQueryClient()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token.trim()) { setError('Token cannot be empty'); return }
    setError('')
    setChecking(true)
    try {
      // Temporarily store the token and let the status endpoint validate it.
      localStorage.setItem(TOKEN_STORAGE_KEY, token.trim())
      const status = await api.getAuthStatus()
      if (status.authenticated) {
        qc.invalidateQueries({ queryKey: ['auth-status'] })
        onSuccess()
      } else {
        localStorage.removeItem(TOKEN_STORAGE_KEY)
        setError('Invalid token. Please check and try again.')
      }
    } catch {
      localStorage.removeItem(TOKEN_STORAGE_KEY)
      setError('Could not verify token. Please try again.')
    } finally {
      setChecking(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 flex items-center justify-center p-4">
      <div className="w-full max-w-sm bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm p-8">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-1">DNS Tester</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
          API token authentication is enabled. Enter your API token to continue.
        </p>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              API Token
            </label>
            <input
              type="password"
              value={token}
              onChange={e => setToken(e.target.value)}
              placeholder="Paste your token here"
              autoFocus
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          {error && (
            <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
          )}

          <button type="submit" disabled={checking} className="w-full btn-primary py-2">
            {checking ? 'Verifying…' : 'Continue'}
          </button>
        </form>
      </div>
    </div>
  )
}
