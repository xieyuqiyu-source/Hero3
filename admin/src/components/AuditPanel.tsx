import { auditLogs, guardrails } from '@/data'

export function AuditPanel() {
  return (
    <article className="panel" id="审计">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">Audit Log</p>
          <h2>操作审计</h2>
        </div>
      </div>

      <ol className="audit-list">
        {auditLogs.map((log) => (
          <li key={log.time}>
            <time>{log.time}</time>
            <span>{log.action}</span>
          </li>
        ))}
      </ol>
    </article>
  )
}

export function GuardrailPanel() {
  return (
    <section className="guardrail-panel">
      {guardrails.map((item) => (
        <div key={item.title}>
          <strong>{item.title}</strong>
          <span>{item.desc}</span>
        </div>
      ))}
    </section>
  )
}
