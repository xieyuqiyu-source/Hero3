import { useMemo, useState } from 'react'
import { FileText } from 'lucide-react'
import { apiDocs, type ApiDocItem } from '@/data/apiDocs'

interface ApiCheckResult {
  ok: boolean
  status?: number
  elapsedMs?: number
  message: string
}

type ApiCheckMap = Record<string, ApiCheckResult | undefined>

function getStatusHint(result: ApiCheckResult | undefined) {
  if (!result) return '未测试'
  if (result.ok) return '可用'
  if (result.status === 404) return '不存在'
  if (result.status === 502) return '未连接'
  if (result.status && result.status >= 400) return '失败'
  return '连接失败'
}

function buildRequestOptions(item: ApiDocItem): RequestInit {
  if (item.method === 'GET') return { method: item.method }
  return {
    method: item.method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(item.sampleBody ?? {}),
  }
}

export default function ApiDiagnosticsPanel() {
  const [results, setResults] = useState<ApiCheckMap>({})
  const [runningId, setRunningId] = useState<string | null>(null)

  const summary = useMemo(() => {
    const tested = Object.values(results).filter(Boolean)
    const okCount = tested.filter((r) => r?.ok).length
    return `${okCount}/${tested.length || apiDocs.length} 可用`
  }, [results])

  const runCheck = async (item: ApiDocItem) => {
    if (item.destructive) {
      setResults((c) => ({ ...c, [item.id]: { ok: false, message: '高危接口不支持诊断面板测试' } }))
      return
    }
    setRunningId(item.id)
    const start = performance.now()
    try {
      const res = await fetch(item.path, buildRequestOptions(item))
      const text = await res.text()
      setResults((c) => ({
        ...c,
        [item.id]: { ok: res.ok, status: res.status, elapsedMs: Math.round(performance.now() - start), message: res.ok ? 'OK' : text.slice(0, 100) },
      }))
    } catch (err) {
      setResults((c) => ({
        ...c,
        [item.id]: { ok: false, elapsedMs: Math.round(performance.now() - start), message: err instanceof Error ? err.message : '请求失败' },
      }))
    } finally {
      setRunningId(null)
    }
  }

  const runAll = async () => {
    for (const item of apiDocs.filter((a) => !a.destructive)) await runCheck(item)
  }

  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <FileText size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-base font-bold text-[var(--color-text-primary)]">接口诊断</h2>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-xs font-bold text-[var(--color-text-secondary)]">{summary}</span>
          <button
            type="button"
            onClick={runAll}
            disabled={runningId !== null}
            className="px-3 py-1.5 rounded-xl text-xs font-bold border border-[var(--color-accent-border)] text-[var(--color-accent)] bg-[var(--color-accent-light)] cursor-pointer hover:bg-[var(--color-accent)]/15 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {runningId ? '测试中...' : '全部测试'}
          </button>
        </div>
      </div>

      <div className="grid gap-2">
        {apiDocs.map((item) => {
          const result = results[item.id]
          return (
            <div key={item.id} className="flex items-center gap-3 px-3 py-2.5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
              <span className={`
                inline-flex min-w-[44px] justify-center px-1.5 py-0.5 rounded-md text-[10px] font-black text-white
                ${item.method === 'GET' ? 'bg-indigo-600' : item.method === 'POST' ? 'bg-teal-700' : item.method === 'PUT' ? 'bg-violet-700' : 'bg-red-700'}
              `}>
                {item.method}
              </span>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <strong className="text-xs text-[var(--color-text-primary)]">{item.title}</strong>
                  <code className="text-[10px] text-[var(--color-text-muted)] truncate">{item.path}</code>
                </div>
              </div>
              <div className="flex items-center gap-2 flex-shrink-0">
                {result && (
                  <span className={`text-[10px] font-bold ${result.ok ? 'text-emerald-600' : 'text-red-600'}`}>
                    {getStatusHint(result)}
                  </span>
                )}
                <button
                  type="button"
                  onClick={() => runCheck(item)}
                  disabled={runningId !== null}
                  className="px-2 py-1 rounded-lg text-[10px] font-bold border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {item.destructive ? '高危' : runningId === item.id ? '...' : '测试'}
                </button>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
