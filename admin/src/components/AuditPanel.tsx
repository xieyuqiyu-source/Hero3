import { ShieldCheck, Lock } from 'lucide-react'

const auditLogs = [
  { time: '15:20', action: 'admin 调整 demo-player 粮食 +5000' },
  { time: '15:06', action: 'system 生成测试地图目标' },
  { time: '14:48', action: 'admin 刷新战斗规则配置' },
]

const guardrails = [
  { title: '权限隔离', desc: '后续所有高危操作必须接入管理员身份和二次确认。' },
  { title: '审计优先', desc: '资源调整、封禁、配置刷新必须写入不可篡改日志。' },
  { title: '测试服优先', desc: '当前后台只面向开发环境，不连接正式服数据。' },
]

export function AuditPanel() {
  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <ShieldCheck size={16} className="text-[var(--color-accent)]" />
        <h2 className="text-base font-bold text-[var(--color-text-primary)]">操作日志</h2>
      </div>
      <div className="grid gap-2">
        {auditLogs.map((log) => (
          <div key={log.time + log.action} className="flex items-center gap-3 px-3 py-2.5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <time className="text-[11px] font-mono text-[var(--color-text-muted)] flex-shrink-0">{log.time}</time>
            <span className="text-xs text-[var(--color-text-secondary)]">{log.action}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

export function GuardrailPanel() {
  return (
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-[var(--shadow-panel)] p-4">
      <div className="flex items-center gap-2 mb-3">
        <Lock size={16} className="text-[var(--color-accent)]" />
        <h2 className="text-base font-bold text-[var(--color-text-primary)]">安全边界</h2>
      </div>
      <div className="grid gap-2 sm:grid-cols-3">
        {guardrails.map((g) => (
          <div key={g.title} className="p-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
            <strong className="block text-sm text-[var(--color-text-primary)] mb-1">{g.title}</strong>
            <span className="text-xs text-[var(--color-text-secondary)]">{g.desc}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
