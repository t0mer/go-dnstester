import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import type { Config } from '../types'

interface Props {
  config: Config
}

export function FQDNConfig({ config }: Props) {
  const qc = useQueryClient()
  const [fqdns, setFqdns] = useState<string[]>(config.fqdns)
  const [newFqdn, setNewFqdn] = useState('')

  const save = useMutation({
    mutationFn: () => api.updateConfig({ ...config, fqdns }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['config'] }),
  })

  const add = () => {
    const v = newFqdn.trim().toLowerCase()
    if (!v || fqdns.includes(v)) return
    setFqdns(f => [...f, v])
    setNewFqdn('')
  }

  const remove = (i: number) => setFqdns(f => f.filter((_, idx) => idx !== i))

  return (
    <div className="bg-white border border-gray-200 rounded-lg">
      <div className="px-5 py-4 border-b border-gray-200 flex items-center justify-between">
        <h2 className="text-base font-semibold text-gray-900">Query FQDNs</h2>
        <button
          onClick={() => save.mutate()}
          disabled={save.isPending}
          className="btn-primary"
        >
          {save.isPending ? 'Saving…' : 'Save'}
        </button>
      </div>

      <ul className="divide-y divide-gray-100">
        {fqdns.map((fqdn, i) => (
          <li key={i} className="flex items-center gap-3 px-5 py-3">
            <span className="flex-1 text-sm font-mono text-gray-700">{fqdn}</span>
            <button
              onClick={() => remove(i)}
              className="text-gray-300 hover:text-red-500 transition-colors text-lg leading-none"
            >
              ×
            </button>
          </li>
        ))}
      </ul>

      <div className="px-5 py-4 border-t border-gray-100 flex items-center gap-2">
        <input
          type="text"
          placeholder="example.com"
          value={newFqdn}
          onChange={e => setNewFqdn(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && add()}
          className="input flex-1"
        />
        <button onClick={add} className="btn-primary">Add</button>
      </div>
    </div>
  )
}
