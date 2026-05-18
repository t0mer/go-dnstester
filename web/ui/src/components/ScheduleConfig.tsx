import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import type { Config, ScheduledScan, ScheduleType, RunSummary } from '../types'

const DAY_SHORT = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
const DAY_LONG  = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

// ── helpers ─────────────────────────────────────────────────────────────────

function describeSchedule(s: ScheduledScan): string {
  switch (s.type) {
    case 'interval': {
      const mins = s.interval_minutes ?? 0
      if (mins < 60) return `Every ${mins} minute${mins !== 1 ? 's' : ''}`
      const h = Math.floor(mins / 60), m = mins % 60
      return m > 0 ? `Every ${h}h ${m}m` : `Every ${h} hour${h !== 1 ? 's' : ''}`
    }
    case 'daily':
      return `Daily at ${s.time_of_day ?? '00:00'}`
    case 'weekdays': {
      const days = (s.weekdays ?? []).map(d => DAY_SHORT[d]).join(', ')
      return `${days || '—'} at ${s.time_of_day ?? '00:00'}`
    }
    case 'weekly':
      return `Every ${DAY_LONG[s.weekday ?? 0]} at ${s.time_of_day ?? '00:00'}`
    case 'monthly':
      return `Monthly on day ${s.day_of_month ?? 1} at ${s.time_of_day ?? '00:00'}`
    case 'once':
      return s.run_at
        ? `Once · ${new Date(s.run_at).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })}`
        : 'Once (not configured)'
  }
}

function computeNextRun(s: ScheduledScan): Date | null {
  const now = new Date()
  const [h, m] = (s.time_of_day ?? '00:00').split(':').map(Number)

  switch (s.type) {
    case 'interval': {
      const mins = s.interval_minutes ?? 0
      if (mins <= 0) return null
      const next = new Date(now.getTime() + mins * 60_000)
      return next
    }
    case 'daily': {
      const t = new Date(); t.setHours(h, m, 0, 0)
      if (t <= now) t.setDate(t.getDate() + 1)
      return t
    }
    case 'weekdays': {
      const days = s.weekdays ?? []
      for (let i = 0; i < 8; i++) {
        const t = new Date(); t.setDate(t.getDate() + i); t.setHours(h, m, 0, 0)
        if (days.includes(t.getDay()) && t > now) return t
      }
      return null
    }
    case 'weekly': {
      for (let i = 0; i < 8; i++) {
        const t = new Date(); t.setDate(t.getDate() + i); t.setHours(h, m, 0, 0)
        if (t.getDay() === (s.weekday ?? 0) && t > now) return t
      }
      return null
    }
    case 'monthly': {
      const t = new Date(); t.setDate(s.day_of_month ?? 1); t.setHours(h, m, 0, 0)
      if (t <= now) t.setMonth(t.getMonth() + 1)
      return t
    }
    case 'once':
      return s.run_at ? new Date(s.run_at) : null
  }
}

function relTime(d: Date | null): string {
  if (!d) return '—'
  const diff = d.getTime() - Date.now()
  if (diff < 0) return 'overdue'
  const mins = Math.round(diff / 60_000)
  if (mins < 60) return `in ${mins}m`
  const hours = Math.round(diff / 3_600_000)
  if (hours < 24) return `in ${hours}h`
  const days = Math.round(diff / 86_400_000)
  return `in ${days}d`
}

function newId() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  // fallback for non-secure contexts (HTTP over IP)
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = (Math.random() * 16) | 0
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16)
  })
}

// ── form state ───────────────────────────────────────────────────────────────

interface FormState {
  name: string
  type: ScheduleType
  intervalValue: number
  intervalUnit: 'minutes' | 'hours'
  timeOfDay: string
  weekdays: number[]
  weekday: number
  dayOfMonth: number
  runAt: string  // "YYYY-MM-DDTHH:MM" for datetime-local
}

const DEFAULT_FORM: FormState = {
  name: '',
  type: 'daily',
  intervalValue: 1,
  intervalUnit: 'hours',
  timeOfDay: '02:00',
  weekdays: [1, 2, 3, 4, 5],
  weekday: 1,
  dayOfMonth: 1,
  runAt: '',
}

function formToScan(f: FormState, id: string): ScheduledScan {
  const base = { id, name: f.name, enabled: true, type: f.type }
  switch (f.type) {
    case 'interval':
      return { ...base, interval_minutes: f.intervalUnit === 'hours' ? f.intervalValue * 60 : f.intervalValue }
    case 'daily':
      return { ...base, time_of_day: f.timeOfDay }
    case 'weekdays':
      return { ...base, weekdays: f.weekdays, time_of_day: f.timeOfDay }
    case 'weekly':
      return { ...base, weekday: f.weekday, time_of_day: f.timeOfDay }
    case 'monthly':
      return { ...base, day_of_month: f.dayOfMonth, time_of_day: f.timeOfDay }
    case 'once':
      return { ...base, run_at: f.runAt ? new Date(f.runAt).toISOString() : '' }
  }
}

function scanToForm(s: ScheduledScan): FormState {
  const mins = s.interval_minutes ?? 60
  const isWholeHours = mins >= 60 && mins % 60 === 0
  return {
    name: s.name,
    type: s.type,
    intervalValue: isWholeHours ? mins / 60 : mins,
    intervalUnit: isWholeHours ? 'hours' : 'minutes',
    timeOfDay: s.time_of_day ?? '02:00',
    weekdays: s.weekdays ?? [1, 2, 3, 4, 5],
    weekday: s.weekday ?? 1,
    dayOfMonth: s.day_of_month ?? 1,
    runAt: s.run_at ? new Date(s.run_at).toISOString().slice(0, 16) : '',
  }
}

// ── sub-components ───────────────────────────────────────────────────────────

function WeekdayPicker({ value, onChange }: { value: number[]; onChange: (v: number[]) => void }) {
  const toggle = (d: number) =>
    onChange(value.includes(d) ? value.filter(x => x !== d) : [...value, d])
  return (
    <div className="flex gap-1 flex-wrap">
      {DAY_SHORT.map((label, d) => (
        <button
          key={d}
          type="button"
          onClick={() => toggle(d)}
          className={`px-2.5 py-1 rounded text-sm font-medium border transition-colors ${
            value.includes(d)
              ? 'bg-blue-600 text-white border-blue-600'
              : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'
          }`}
        >
          {label}
        </button>
      ))}
    </div>
  )
}

function ScheduleFormFields({ form, setForm }: { form: FormState; setForm: (f: FormState) => void }) {
  const upd = (patch: Partial<FormState>) => setForm({ ...form, ...patch })

  return (
    <div className="space-y-3">
      <div className="flex gap-3 flex-wrap">
        <div>
          <label className="block text-xs text-gray-500 mb-1">Name</label>
          <input
            className="input w-48"
            placeholder="My schedule"
            value={form.name}
            onChange={e => upd({ name: e.target.value })}
          />
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">Type</label>
          <select
            className="input"
            value={form.type}
            onChange={e => upd({ type: e.target.value as ScheduleType })}
          >
            <option value="interval">Interval (every N min/hours)</option>
            <option value="daily">Daily</option>
            <option value="weekdays">Specific weekdays</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
            <option value="once">Once (specific date & time)</option>
          </select>
        </div>
      </div>

      {form.type === 'interval' && (
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-600">Every</span>
          <input
            type="number"
            min={1}
            className="input w-20"
            value={form.intervalValue}
            onChange={e => upd({ intervalValue: Math.max(1, Number(e.target.value)) })}
          />
          <select className="input" value={form.intervalUnit} onChange={e => upd({ intervalUnit: e.target.value as 'minutes' | 'hours' })}>
            <option value="minutes">Minutes</option>
            <option value="hours">Hours</option>
          </select>
        </div>
      )}

      {(form.type === 'daily' || form.type === 'weekdays' || form.type === 'weekly' || form.type === 'monthly') && (
        <div className="flex items-center gap-3 flex-wrap">
          {form.type === 'weekdays' && (
            <WeekdayPicker value={form.weekdays} onChange={weekdays => upd({ weekdays })} />
          )}
          {form.type === 'weekly' && (
            <select className="input" value={form.weekday} onChange={e => upd({ weekday: Number(e.target.value) })}>
              {DAY_LONG.map((d, i) => <option key={i} value={i}>{d}</option>)}
            </select>
          )}
          {form.type === 'monthly' && (
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-600">On day</span>
              <input
                type="number" min={1} max={31} className="input w-20"
                value={form.dayOfMonth}
                onChange={e => upd({ dayOfMonth: Math.min(31, Math.max(1, Number(e.target.value))) })}
              />
              <span className="text-sm text-gray-600">of each month</span>
            </div>
          )}
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-600">at</span>
            <input
              type="time" className="input"
              value={form.timeOfDay}
              onChange={e => upd({ timeOfDay: e.target.value })}
            />
          </div>
        </div>
      )}

      {form.type === 'once' && (
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-600">Run on</span>
          <input
            type="datetime-local" className="input"
            value={form.runAt}
            onChange={e => upd({ runAt: e.target.value })}
          />
        </div>
      )}
    </div>
  )
}

// ── main component ───────────────────────────────────────────────────────────

interface Props {
  config: Config
  history: RunSummary[]
}

export function ScheduleConfig({ config, history }: Props) {
  const qc = useQueryClient()
  const [editingId, setEditingId] = useState<string | null>(null)  // null = not editing, 'new' = adding
  const [form, setForm] = useState<FormState>(DEFAULT_FORM)

  const save = useMutation({
    mutationFn: (schedules: ScheduledScan[]) =>
      api.updateConfig({ ...config, schedules }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['config'] })
      setEditingId(null)
    },
  })

  const schedules = config.schedules ?? []

  const startAdd = () => {
    setForm(DEFAULT_FORM)
    setEditingId('new')
  }

  const startEdit = (s: ScheduledScan) => {
    setForm(scanToForm(s))
    setEditingId(s.id)
  }

  const commitForm = () => {
    if (!form.name.trim()) return
    const id = editingId === 'new' ? newId() : editingId!
    const scan = formToScan(form, id)
    const updated = editingId === 'new'
      ? [...schedules, scan]
      : schedules.map(s => s.id === id ? scan : s)
    save.mutate(updated)
  }

  const toggleEnabled = (id: string) => {
    save.mutate(schedules.map(s => s.id === id ? { ...s, enabled: !s.enabled } : s))
  }

  const remove = (id: string) => {
    save.mutate(schedules.filter(s => s.id !== id))
  }

  // Find last run for a schedule from history
  const lastRunFor = (id: string) =>
    history.filter(r => r.schedule_id === id).sort((a, b) =>
      new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
    )[0]

  return (
    <div className="bg-white border border-gray-200 rounded-lg">
      <div className="px-5 py-4 border-b border-gray-200 flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold text-gray-900">Scheduled Scans</h2>
          <p className="text-xs text-gray-500 mt-0.5">
            Automatically run tests on a schedule. Results are tagged and available in History &amp; Compare.
          </p>
        </div>
        <button onClick={startAdd} className="btn-primary">+ Add schedule</button>
      </div>

      {/* Add form */}
      {editingId === 'new' && (
        <div className="px-5 py-4 bg-blue-50 border-b border-blue-100">
          <p className="text-sm font-semibold text-blue-800 mb-3">New schedule</p>
          <ScheduleFormFields form={form} setForm={setForm} />
          <div className="flex gap-2 mt-3">
            <button onClick={commitForm} disabled={save.isPending || !form.name.trim()} className="btn-primary">
              {save.isPending ? 'Saving…' : 'Add'}
            </button>
            <button onClick={() => setEditingId(null)} className="btn-secondary">Cancel</button>
          </div>
        </div>
      )}

      {schedules.length === 0 && editingId !== 'new' && (
        <p className="text-sm text-gray-400 px-5 py-6 text-center">
          No schedules yet. Click <strong>+ Add schedule</strong> to create one.
        </p>
      )}

      <ul className="divide-y divide-gray-100">
        {schedules.map(sc => {
          const isEditing = editingId === sc.id
          const lastRun = lastRunFor(sc.id)
          const nextRun = sc.enabled ? computeNextRun(sc) : null

          return (
            <li key={sc.id}>
              {isEditing ? (
                <div className="px-5 py-4 bg-yellow-50 border-b border-yellow-100">
                  <p className="text-sm font-semibold text-yellow-800 mb-3">Editing "{sc.name}"</p>
                  <ScheduleFormFields form={form} setForm={setForm} />
                  <div className="flex gap-2 mt-3">
                    <button onClick={commitForm} disabled={save.isPending || !form.name.trim()} className="btn-primary">
                      {save.isPending ? 'Saving…' : 'Save'}
                    </button>
                    <button onClick={() => setEditingId(null)} className="btn-secondary">Cancel</button>
                  </div>
                </div>
              ) : (
                <div className="px-5 py-3 flex items-center gap-4">
                  <input
                    type="checkbox"
                    checked={sc.enabled}
                    onChange={() => toggleEnabled(sc.id)}
                    className="h-4 w-4 text-blue-600 rounded border-gray-300 flex-shrink-0"
                  />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900 truncate">{sc.name}</p>
                    <p className="text-xs text-gray-500">{describeSchedule(sc)}</p>
                  </div>
                  <div className="text-right text-xs text-gray-400 hidden sm:block flex-shrink-0">
                    {nextRun && <p>Next: <span className="text-gray-600">{relTime(nextRun)}</span></p>}
                    {lastRun && (
                      <p>Last: <span className="text-gray-600">
                        {new Date(lastRun.started_at).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })}
                      </span></p>
                    )}
                  </div>
                  <div className="flex items-center gap-1 flex-shrink-0">
                    <button onClick={() => startEdit(sc)} className="btn-secondary text-xs px-2 py-1">Edit</button>
                    <button
                      onClick={() => remove(sc.id)}
                      className="text-xs px-2 py-1 rounded border border-red-200 text-red-600 hover:bg-red-50 transition-colors"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              )}
            </li>
          )
        })}
      </ul>
    </div>
  )
}
