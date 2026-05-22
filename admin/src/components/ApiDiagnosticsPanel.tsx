import { useMemo, useState } from 'react'
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
  if (result.status === 404) return '接口不存在'
  if (result.status === 502) return '后端未连接'
  if (result.status && result.status >= 400) return '请求失败'
  return '连接失败'
}

function getResultClass(result: ApiCheckResult | undefined) {
  if (!result) return 'idle'
  return result.ok ? 'ok' : 'fail'
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
    const okCount = tested.filter((result) => result?.ok).length
    return `${okCount}/${tested.length || apiDocs.length} 可用`
  }, [results])

  const runCheck = async (item: ApiDocItem) => {
    if (item.destructive) {
      setResults((current) => ({
        ...current,
        [item.id]: {
          ok: false,
          message: '高危接口不支持在诊断面板直接测试，请在业务面板二次确认后操作。',
        },
      }))
      return
    }

    setRunningId(item.id)
    const startedAt = performance.now()

    try {
      const response = await fetch(item.path, buildRequestOptions(item))
      const text = await response.text()
      const elapsedMs = Math.round(performance.now() - startedAt)
      setResults((current) => ({
        ...current,
        [item.id]: {
          ok: response.ok,
          status: response.status,
          elapsedMs,
          message: response.ok ? '接口返回正常' : text.slice(0, 120) || response.statusText,
        },
      }))
    } catch (error) {
      setResults((current) => ({
        ...current,
        [item.id]: {
          ok: false,
          elapsedMs: Math.round(performance.now() - startedAt),
          message: error instanceof Error ? error.message : '请求未发出',
        },
      }))
    } finally {
      setRunningId(null)
    }
  }

  const runAll = async () => {
    for (const item of apiDocs.filter((api) => !api.destructive)) {
      await runCheck(item)
    }
  }

  return (
    <article className="panel api-panel" id="接口">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">API Diagnostics</p>
          <h2>接口文档与诊断</h2>
        </div>
        <button type="button" onClick={runAll} disabled={runningId !== null}>
          {runningId ? '测试中' : '全部测试'}
        </button>
      </div>

      <div className="api-summary">
        <span>代理入口：/api、/healthz</span>
        <strong>{summary}</strong>
      </div>

      <div className="api-list">
        {apiDocs.map((item) => {
          const result = results[item.id]
          return (
            <section className="api-row" key={item.id}>
              <div className="api-main">
                <div className="api-title-line">
                  <span className={`method-badge ${item.method.toLowerCase()}`}>{item.method}</span>
                  <strong>{item.title}</strong>
                  <em>{item.status}</em>
                </div>
                <code>{item.path}</code>
                <p>{item.desc}</p>
                <small>前端使用：{item.usedBy}</small>
              </div>

              <div className="api-actions">
                <span className={`api-status ${getResultClass(result)}`}>{getStatusHint(result)}</span>
                {result && (
                  <small>
                    {result.status ? `HTTP ${result.status}` : 'Network'} · {result.elapsedMs}ms
                  </small>
                )}
                <button type="button" onClick={() => runCheck(item)} disabled={runningId !== null}>
                  {item.destructive ? '高危' : runningId === item.id ? '请求中' : '测试'}
                </button>
              </div>

              {result && !result.ok && <div className="api-error">{result.message}</div>}
            </section>
          )
        })}
      </div>
    </article>
  )
}
