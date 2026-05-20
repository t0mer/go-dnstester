import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, TOKEN_STORAGE_KEY } from '../api/client'

export function AuthSettings() {
  const qc = useQueryClient()

  const { data: status, isLoading } = useQuery({
    queryKey: ['auth-status'],
    queryFn: api.getAuthStatus,
    staleTime: 0,
  })

  // --- auth enable/disable form state ---
  const [enabled, setEnabled] = useState<boolean | null>(null)
  const [apiTokenEnabled, setApiTokenEnabled] = useState<boolean | null>(null)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [formError, setFormError] = useState('')

  const effectiveEnabled = enabled ?? status?.auth_enabled ?? false
  const effectiveTokenEnabled = apiTokenEnabled ?? status?.api_token_enabled ?? false

  // --- token display (shown once after generation) ---
  const [newToken, setNewToken] = useState<string | null>(null)
  const [tokenCopied, setTokenCopied] = useState(false)

  const browserHasToken = !!localStorage.getItem(TOKEN_STORAGE_KEY)

  const saveAuth = useMutation({
    mutationFn: () => {
      setFormError('')
      if (effectiveEnabled) {
        if (!username.trim()) { setFormError('Username cannot be empty'); return Promise.reject() }
        if (password && password.length < 8) { setFormError('Password must be at least 8 characters'); return Promise.reject() }
        if (password !== confirmPassword) { setFormError('Passwords do not match'); return Promise.reject() }
        if (!status?.has_credentials && !password) { setFormError('A password is required to enable authentication'); return Promise.reject() }
      }
      return api.updateAuthSettings({
        enabled: effectiveEnabled,
        username: username || status?.username || '',
        password,
        api_token_enabled: effectiveTokenEnabled,
      })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['auth-status'] })
      setPassword('')
      setConfirmPassword('')
    },
    onError: (e) => {
      if (e instanceof Error && e.message) setFormError(e.message)
    },
  })

  const generateToken = useMutation({
    mutationFn: api.generateToken,
    onSuccess: (data) => {
      setNewToken(data.token)
      setTokenCopied(false)
      // Store in localStorage so the browser UI sends it automatically on every
      // API request — this is what lets the app keep working after token auth
      // is enabled without requiring the user to paste it on every visit.
      localStorage.setItem(TOKEN_STORAGE_KEY, data.token)
      qc.invalidateQueries({ queryKey: ['auth-status'] })
    },
  })

  const revokeToken = useMutation({
    mutationFn: api.revokeToken,
    onSuccess: () => {
      setNewToken(null)
      // Clear the stored token so the UI stops sending it. The server also
      // disables APITokenEnabled on revoke so no auth is required until the
      // user explicitly re-enables it and generates a new token.
      localStorage.removeItem(TOKEN_STORAGE_KEY)
      setApiTokenEnabled(false)
      qc.invalidateQueries({ queryKey: ['auth-status'] })
    },
  })

  const copyToken = () => {
    if (!newToken) return
    navigator.clipboard.writeText(newToken).then(() => setTokenCopied(true))
  }

  if (isLoading) return null

  return (
    <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg">
      <div className="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
        <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">Authentication</h2>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
          Protect the UI and API with a username/password login and optional API tokens.
        </p>
      </div>

      <div className="px-5 py-4 space-y-6">

        {/* Enable toggle */}
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Require login</p>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
              Require a username and password to access the UI and API.
            </p>
          </div>
          <button
            onClick={() => setEnabled(!effectiveEnabled)}
            role="switch"
            aria-checked={effectiveEnabled}
            className={`mt-0.5 relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 ${
              effectiveEnabled ? 'bg-blue-600' : 'bg-gray-200 dark:bg-gray-600'
            }`}
          >
            <span className="sr-only">Enable authentication</span>
            <span className={`inline-block h-4 w-4 rounded-full bg-white shadow transform transition-transform ${effectiveEnabled ? 'translate-x-6' : 'translate-x-1'}`} />
          </button>
        </div>

        {/* Credentials */}
        {effectiveEnabled && (
          <div className="space-y-3 pl-1">
            <div>
              <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Username</label>
              <input
                type="text"
                autoComplete="username"
                value={username || status?.username || ''}
                onChange={e => setUsername(e.target.value)}
                placeholder="admin"
                className="w-full max-w-xs px-3 py-1.5 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                {status?.has_credentials ? 'New password' : 'Password'}
                {status?.has_credentials && <span className="ml-1 font-normal text-gray-400">(leave blank to keep current)</span>}
              </label>
              <input
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="Min. 8 characters"
                className="w-full max-w-xs px-3 py-1.5 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            {password && (
              <div>
                <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Confirm password</label>
                <input
                  type="password"
                  autoComplete="new-password"
                  value={confirmPassword}
                  onChange={e => setConfirmPassword(e.target.value)}
                  className="w-full max-w-xs px-3 py-1.5 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            )}
          </div>
        )}

        {/* API Token section */}
        <div className="border-t border-gray-100 dark:border-gray-800 pt-5 space-y-4">
          <div className="flex items-start justify-between gap-4">
            <div>
              <p className="text-sm font-medium text-gray-900 dark:text-gray-100">API token authentication</p>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                Require a Bearer token for external API access.{' '}
                <code className="font-mono">Authorization: Bearer &lt;token&gt;</code>
              </p>
            </div>
            <button
              onClick={() => setApiTokenEnabled(!effectiveTokenEnabled)}
              role="switch"
              aria-checked={effectiveTokenEnabled}
              className={`mt-0.5 relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 ${
                effectiveTokenEnabled ? 'bg-blue-600' : 'bg-gray-200 dark:bg-gray-600'
              }`}
            >
              <span className="sr-only">Enable API token auth</span>
              <span className={`inline-block h-4 w-4 rounded-full bg-white shadow transform transition-transform ${effectiveTokenEnabled ? 'translate-x-6' : 'translate-x-1'}`} />
            </button>
          </div>

          {effectiveTokenEnabled && (
            <div className="pl-1 space-y-3">

              {/* Explain how the browser handles token auth */}
              {!effectiveEnabled && (
                <p className="text-xs text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg px-3 py-2">
                  When login is not required, the browser automatically uses the generated token for its own API calls. External clients must also supply the token.
                  {browserHasToken && !newToken && (
                    <span className="block mt-0.5 font-medium">This browser has a stored token and will authenticate automatically.</span>
                  )}
                </p>
              )}

              {/* Token display (shown once after generation) */}
              {newToken && (
                <div className="rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 p-3 space-y-2">
                  <p className="text-xs font-medium text-amber-800 dark:text-amber-300">
                    Token generated. Copy it now — it won't be shown again. This browser will use it automatically.
                  </p>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 font-mono text-xs break-all text-amber-900 dark:text-amber-200 bg-white dark:bg-gray-800 px-2 py-1.5 rounded border border-amber-200 dark:border-amber-700">
                      {newToken}
                    </code>
                    <button
                      onClick={copyToken}
                      className="text-xs px-2 py-1.5 rounded border border-amber-300 dark:border-amber-600 text-amber-700 dark:text-amber-300 hover:bg-amber-100 dark:hover:bg-amber-900/40 whitespace-nowrap"
                    >
                      {tokenCopied ? 'Copied!' : 'Copy'}
                    </button>
                  </div>
                </div>
              )}

              <div className="flex flex-wrap items-center gap-2">
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  {status?.has_token ? 'A token is active.' : 'No token configured.'}
                </span>
                <button
                  onClick={() => generateToken.mutate()}
                  disabled={generateToken.isPending}
                  className="btn-secondary text-xs"
                >
                  {generateToken.isPending ? 'Generating…' : status?.has_token ? 'Regenerate token' : 'Generate token'}
                </button>
                {status?.has_token && (
                  <button
                    onClick={() => revokeToken.mutate()}
                    disabled={revokeToken.isPending}
                    className="text-xs px-3 py-1.5 rounded-lg border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                    title="Revokes the token and disables API token auth"
                  >
                    {revokeToken.isPending ? 'Revoking…' : 'Revoke & disable'}
                  </button>
                )}
              </div>
            </div>
          )}
        </div>

        {formError && (
          <p className="text-sm text-red-600 dark:text-red-400">{formError}</p>
        )}
        {saveAuth.isError && !formError && (
          <p className="text-sm text-red-600 dark:text-red-400">
            {saveAuth.error instanceof Error ? saveAuth.error.message : 'Save failed'}
          </p>
        )}

        <div className="flex justify-end pt-1">
          <button
            onClick={() => saveAuth.mutate()}
            disabled={saveAuth.isPending}
            className="btn-primary"
          >
            {saveAuth.isPending ? 'Saving…' : 'Save'}
          </button>
        </div>

      </div>
    </div>
  )
}
