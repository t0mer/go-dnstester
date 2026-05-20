import { useState, useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import type { Config, DNSServer } from '../types'

interface Props {
  config: Config
}

export function ServerConfig({ config }: Props) {
  const qc = useQueryClient()
  const [servers, setServers] = useState<DNSServer[]>(config.servers)
  const [newName, setNewName] = useState('')
  const [newAddr, setNewAddr] = useState('')
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
    setServers(s => [...s, { name: newName.trim(), address: newAddr.trim(), enabled: true }])
    setNewName('')
    setNewAddr('')
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
            <span className="flex-1 text-sm font-medium text-gray-900 dark:text-gray-100">{srv.name}</span>
            <span className="text-sm text-gray-400 dark:text-gray-500 font-mono">{srv.address}</span>
            <button
              onClick={() => remove(i)}
              className="text-gray-300 dark:text-gray-600 hover:text-red-500 dark:hover:text-red-400 transition-colors text-lg leading-none"
            >
              ×
            </button>
          </li>
        ))}
      </ul>

      <div className="px-4 sm:px-5 py-4 border-t border-gray-100 dark:border-gray-700 flex flex-col sm:flex-row gap-2">
        <input type="text" placeholder="Name" value={newName} onChange={e => setNewName(e.target.value)} className="input w-full sm:flex-1" />
        <input
          type="text" placeholder="IP / Address" value={newAddr}
          onChange={e => setNewAddr(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && addServer()}
          className="input w-full sm:flex-1"
        />
        <button onClick={addServer} className="btn-primary w-full sm:w-auto">Add</button>
      </div>
    </div>
  )
}
