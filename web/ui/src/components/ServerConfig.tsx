import { useState, useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import type { Config, DNSServer } from '../types'

interface Props {
  config: Config
}

const PROTOCOL_OPTIONS = [
  { value: 'udp', label: 'UDP/53' },
  { value: 'dot', label: 'DoT (TLS/853)' },
  { value: 'doh', label: 'DoH (HTTPS)' },
]

const PROTOCOL_BADGE: Record<string, string> = {
  dot: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
  doh: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
}

function addrPlaceholder(protocol: string) {
  if (protocol === 'doh') return 'https://dns.example.com/dns-query'
  if (protocol === 'dot') return 'IP or hostname (e.g. 1.1.1.1)'
  return 'IP / Address'
}

function protocolLabel(srv: DNSServer) {
  const p = srv.protocol
  if (!p || p === 'udp') return null
  return (
    <span className={`text-[10px] font-semibold uppercase px-1.5 py-0.5 rounded ${PROTOCOL_BADGE[p] ?? 'bg-gray-100 text-gray-500'}`}>
      {p}
    </span>
  )
}

export function ServerConfig({ config }: Props) {
  const qc = useQueryClient()
  const [servers, setServers] = useState<DNSServer[]>(config.servers)
  const [newName, setNewName] = useState('')
  const [newAddr, setNewAddr] = useState('')
  const [newProtocol, setNewProtocol] = useState('udp')
  const fileRef = useRef<HTMLInputElement>(null)

  const save = useMutation({
    mutationFn: () => api.updateConfig({ ...config, servers }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['config'] }),
  })

  const backup = useMutation({ mutationFn: api.backup })
  const restore = useMutation({
    mutationFn: api.restore,
    onSuccess: (cfg) => { setServers(cfg.servers); qc.invalidateQueries({ queryKey: ['config'] }) },
  })

  const handleImport = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const text = await file.text()
    const cfg = await api.importConfig(text)
    setServers(cfg.servers)
    qc.invalidateQueries({ queryKey: ['config'] })
    e.target.value = ''
  }

  const handleExport = async () => {
    const blob = await api.exportConfig()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'dnstester-config.json'
    a.click()
    URL.revokeObjectURL(url)
  }

  const addServer = () => {
    if (!newName.trim() || !newAddr.trim()) return
    const srv: DNSServer = {
      name: newName.trim(),
      address: newAddr.trim(),
      enabled: true,
    }
    if (newProtocol !== 'udp') srv.protocol = newProtocol
    setServers(s => [...s, srv])
    setNewName('')
    setNewAddr('')
    setNewProtocol('udp')
  }

  const toggle = (i: number) =>
    setServers(s => s.map((srv, idx) => idx === i ? { ...srv, enabled: !srv.enabled } : srv))

  const remove = (i: number) => setServers(s => s.filter((_, idx) => idx !== i))

  return (
    <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg">
      <div className="px-5 py-4 border-b border-gray-200 dark:border-gray-700 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">DNS Servers</h2>
        <div className="flex items-center gap-2 flex-wrap">
          <button onClick={() => backup.mutate()} className="btn-secondary">Backup</button>
          <button onClick={() => restore.mutate()} className="btn-secondary">Restore</button>
          <button onClick={handleExport} className="btn-secondary">Export</button>
          <button onClick={() => fileRef.current?.click()} className="btn-secondary">Import</button>
          <input ref={fileRef} type="file" accept=".json" className="hidden" onChange={handleImport} />
          <button onClick={() => save.mutate()} disabled={save.isPending} className="btn-primary">
            {save.isPending ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>

      <ul className="divide-y divide-gray-100 dark:divide-gray-700">
        {servers.map((srv, i) => (
          <li key={i} className="flex items-center gap-3 px-5 py-3">
            <input
              type="checkbox"
              checked={srv.enabled}
              onChange={() => toggle(i)}
              className="h-4 w-4 text-blue-600 rounded border-gray-300 dark:border-gray-600"
            />
            <div className="flex-1 min-w-0 flex items-center gap-2">
              <span className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{srv.name}</span>
              {protocolLabel(srv)}
            </div>
            <span className="text-sm text-gray-400 dark:text-gray-500 font-mono truncate max-w-[160px] sm:max-w-xs hidden sm:block">
              {srv.address}
            </span>
            <button
              onClick={() => remove(i)}
              className="text-gray-300 dark:text-gray-600 hover:text-red-500 dark:hover:text-red-400 transition-colors text-lg leading-none flex-shrink-0"
            >
              ×
            </button>
          </li>
        ))}
      </ul>

      {/* Add server form */}
      <div className="px-4 sm:px-5 py-4 border-t border-gray-100 dark:border-gray-700 space-y-2">
        <div className="flex flex-col sm:flex-row gap-2">
          <input
            type="text"
            placeholder="Name"
            value={newName}
            onChange={e => setNewName(e.target.value)}
            className="input w-full sm:flex-1"
          />
          <select
            value={newProtocol}
            onChange={e => setNewProtocol(e.target.value)}
            className="input w-full sm:w-auto"
          >
            {PROTOCOL_OPTIONS.map(o => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        </div>
        <div className="flex flex-col sm:flex-row gap-2">
          <input
            type="text"
            placeholder={addrPlaceholder(newProtocol)}
            value={newAddr}
            onChange={e => setNewAddr(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && addServer()}
            className="input w-full sm:flex-1"
          />
          <button onClick={addServer} className="btn-primary w-full sm:w-auto">Add</button>
        </div>
      </div>
    </div>
  )
}
