/* 本文件实现 GM 公告管理面板，支持公告草稿、发布、定时、撤回、归档和删除。 */
import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { Archive, CalendarClock, Megaphone, Pencil, Pin, RefreshCw, Send, Trash2, Undo2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { Announcement, AnnouncementTarget, SaveAnnouncementPayload } from '@/types'

const PAGE_SIZE = 20

const TYPE_OPTIONS = [
  { value: 'system', label: '系统' },
  { value: 'maintenance', label: '维护' },
  { value: 'update', label: '更新' },
  { value: 'activity', label: '活动' },
  { value: 'compensation', label: '补偿' },
  { value: 'emergency', label: '紧急' },
]

const STATUS_OPTIONS = [
  { value: '', label: '全部状态' },
  { value: 'draft', label: '草稿' },
  { value: 'scheduled', label: '定时' },
  { value: 'published', label: '已发布' },
  { value: 'withdrawn', label: '已撤回' },
  { value: 'archived', label: '已归档' },
]

const TARGET_OPTIONS = [
  { value: 'all', label: '全体玩家' },
  { value: 'player_ids', label: '指定玩家' },
  { value: 'account_ids', label: '指定账号' },
  { value: 'factions', label: '指定阵营' },
  { value: 'created_at_range', label: '创建时间段' },
  { value: 'level_range', label: '等级段' },
]

const emptyForm = {
  title: '',
  summary: '',
  content: '',
  type: 'system',
  displayMode: 'center_only',
  pinned: false,
  priority: 0,
  forcePopup: false,
  startsAt: '',
  endsAt: '',
  targetType: 'all',
  targetValue: '',
}

type AnnouncementForm = typeof emptyForm

// AnnouncementAdminPanel 渲染公告管理完整工作台。
export default function AnnouncementAdminPanel() {
  const [items, setItems] = useState<Announcement[]>([])
  const [selected, setSelected] = useState<Announcement | null>(null)
  const [form, setForm] = useState<AnnouncementForm>(emptyForm)
  const [statusFilter, setStatusFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState('')

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const isEditing = Boolean(selected)
  const targetHelp = useMemo(() => targetValueHelp(form.targetType), [form.targetType])

  // loadAnnouncements 按当前筛选条件刷新公告列表。
  const loadAnnouncements = async (nextPage = page) => {
    setLoading(true)
    setMessage('')
    try {
      const result = await adminApi.listAnnouncements({
        type: typeFilter,
        status: statusFilter,
        page: nextPage,
        pageSize: PAGE_SIZE,
      })
      setItems(Array.isArray(result.items) ? result.items : [])
      setPage(result.page)
      setTotal(result.total)
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '公告列表加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadAnnouncements(1)
  }, [statusFilter, typeFilter])

  // pickAnnouncement 把列表中的公告载入编辑表单。
  const pickAnnouncement = (announcement: Announcement) => {
    const target = announcement.targets?.[0]
    setSelected(announcement)
    setForm({
      title: announcement.title,
      summary: announcement.summary,
      content: announcement.content ?? '',
      type: announcement.type || 'system',
      displayMode: announcement.displayMode || 'center_only',
      pinned: announcement.pinned,
      priority: announcement.priority,
      forcePopup: announcement.forcePopup,
      startsAt: toDatetimeLocal(announcement.startsAt),
      endsAt: toDatetimeLocal(announcement.endsAt),
      targetType: target?.type || 'all',
      targetValue: targetToInput(target),
    })
  }

  // resetForm 清空编辑状态并回到新建公告模式。
  const resetForm = () => {
    setSelected(null)
    setForm(emptyForm)
  }

  // buildPayload 把表单转换为后端保存公告请求。
  const buildPayload = (status: string): SaveAnnouncementPayload => ({
    title: form.title.trim(),
    summary: form.summary.trim(),
    content: form.content.trim(),
    type: form.type,
    status,
    displayMode: form.displayMode,
    pinned: form.pinned,
    priority: Number(form.priority) || 0,
    forcePopup: form.forcePopup,
    startsAt: fromDatetimeLocal(form.startsAt),
    endsAt: fromDatetimeLocal(form.endsAt),
    targets: [buildTarget(form.targetType, form.targetValue)],
  })

  // saveAnnouncement 保存草稿或更新当前公告。
  const saveAnnouncement = async (status = 'draft') => {
    if (!form.title.trim() || !form.content.trim()) {
      setMessage('请填写标题和正文')
      return
    }
    setLoading(true)
    setMessage('')
    try {
      if (selected) {
        const next = await adminApi.updateAnnouncement(selected.id, buildPayload(selected.status || status))
        setSelected(next)
        setMessage('公告已保存')
      } else {
        const next = await adminApi.createAnnouncement(buildPayload(status))
        setSelected(next)
        setMessage(status === 'published' ? '公告已创建并发布' : '草稿已保存')
      }
      await loadAnnouncements(1)
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '公告保存失败')
    } finally {
      setLoading(false)
    }
  }

  // handleSubmit 处理表单默认提交，保存为草稿。
  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    void saveAnnouncement('draft')
  }

  // runAction 执行发布、定时、撤回、归档或删除草稿操作。
  const runAction = async (action: 'publish' | 'schedule' | 'withdraw' | 'archive' | 'delete') => {
    if (!selected) {
      setMessage('请先选择或保存公告')
      return
    }
    if (action === 'schedule' && !form.startsAt) {
      setMessage('定时发布需要填写开始时间')
      return
    }
    const labels = { publish: '发布', schedule: '定时发布', withdraw: '撤回', archive: '归档', delete: '删除草稿' }
    if (!window.confirm(`确认${labels[action]}公告「${selected.title}」？`)) return
    setLoading(true)
    setMessage('')
    try {
      if (action === 'publish') setSelected(await adminApi.publishAnnouncement(selected.id))
      if (action === 'schedule') setSelected(await adminApi.scheduleAnnouncement(selected.id, fromDatetimeLocal(form.startsAt)))
      if (action === 'withdraw') setSelected(await adminApi.withdrawAnnouncement(selected.id))
      if (action === 'archive') setSelected(await adminApi.archiveAnnouncement(selected.id))
      if (action === 'delete') {
        await adminApi.deleteAnnouncement(selected.id)
        resetForm()
      }
      setMessage(`公告已${labels[action]}`)
      await loadAnnouncements(1)
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '公告操作失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(360px,0.9fr)_minmax(420px,1.1fr)]">
      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)]">
        <div className="flex flex-wrap items-center gap-2 border-b border-[var(--color-border)] px-4 py-3">
          <Megaphone size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">公告列表</h2>
          <button
            type="button"
            onClick={() => void loadAnnouncements(page)}
            className="ml-auto inline-flex h-8 w-8 items-center justify-center rounded-xl border border-[var(--color-border)] text-[var(--color-text-secondary)]"
            aria-label="刷新公告"
          >
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>
        <div className="grid gap-2 border-b border-[var(--color-border)] p-3 sm:grid-cols-2">
          <select value={typeFilter} onChange={(event) => setTypeFilter(event.target.value)} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs">
            <option value="">全部类型</option>
            {TYPE_OPTIONS.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
          </select>
          <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-xs">
            {STATUS_OPTIONS.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
          </select>
        </div>
        <div className="max-h-[680px] space-y-2 overflow-y-auto p-3">
          {items.length === 0 ? (
            <div className="py-10 text-center text-sm text-[var(--color-text-muted)]">暂无公告</div>
          ) : items.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => pickAnnouncement(item)}
              className={`w-full rounded-xl border p-3 text-left transition-colors ${selected?.id === item.id ? 'border-[var(--color-accent-border)] bg-[var(--color-accent-light)]' : 'border-[var(--color-border)] bg-[var(--color-surface-dim)] hover:border-[var(--color-accent-border)]'}`}
            >
              <div className="flex items-center gap-2">
                <span className="text-sm font-semibold text-[var(--color-text-primary)]">{item.title}</span>
                {item.pinned && <Pin size={13} className="text-amber-500" />}
                <span className="ml-auto rounded bg-[var(--color-surface)] px-1.5 py-0.5 text-[10px] text-[var(--color-text-muted)]">{statusLabel(item.status)}</span>
              </div>
              <p className="mt-1 line-clamp-2 text-xs text-[var(--color-text-secondary)]">{item.summary || item.content}</p>
              <p className="mt-2 text-[10px] text-[var(--color-text-muted)]">{typeLabel(item.type)} · 更新 {formatTime(item.updatedAt)}</p>
            </button>
          ))}
        </div>
        <div className="flex items-center justify-between border-t border-[var(--color-border)] px-3 py-2 text-xs text-[var(--color-text-muted)]">
          <span>共 {total} 条 · 第 {page}/{totalPages} 页</span>
          <div className="flex gap-2">
            <button type="button" disabled={page <= 1 || loading} onClick={() => void loadAnnouncements(Math.max(1, page - 1))} className="rounded-lg border border-[var(--color-border)] px-2 py-1 disabled:opacity-40">上一页</button>
            <button type="button" disabled={page >= totalPages || loading} onClick={() => void loadAnnouncements(Math.min(totalPages, page + 1))} className="rounded-lg border border-[var(--color-border)] px-2 py-1 disabled:opacity-40">下一页</button>
          </div>
        </div>
      </section>

      <form onSubmit={handleSubmit} className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)]">
        <div className="flex flex-wrap items-center gap-2 border-b border-[var(--color-border)] px-4 py-3">
          <Pencil size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">{isEditing ? '编辑公告' : '新建公告'}</h2>
          <button type="button" onClick={resetForm} className="ml-auto rounded-xl border border-[var(--color-border)] px-3 py-1.5 text-xs text-[var(--color-text-secondary)]">新建</button>
        </div>
        <div className="grid gap-3 p-4">
          <input value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} maxLength={160} placeholder="标题" className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]" />
          <input value={form.summary} onChange={(event) => setForm({ ...form, summary: event.target.value })} maxLength={255} placeholder="摘要" className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]" />
          <textarea value={form.content} onChange={(event) => setForm({ ...form, content: event.target.value })} rows={8} placeholder="正文" className="resize-none rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]" />
          <div className="grid gap-3 md:grid-cols-3">
            <select value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value })} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm">
              {TYPE_OPTIONS.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
            </select>
            <select value={form.displayMode} onChange={(event) => setForm({ ...form, displayMode: event.target.value })} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm">
              <option value="center_only">公告中心</option>
              <option value="popup">弹窗</option>
              <option value="banner">横幅预留</option>
            </select>
            <input type="number" value={form.priority} onChange={(event) => setForm({ ...form, priority: Number(event.target.value) })} placeholder="优先级" className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm" />
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            <label className="grid gap-1 text-xs text-[var(--color-text-muted)]">
              开始时间
              <input type="datetime-local" value={form.startsAt} onChange={(event) => setForm({ ...form, startsAt: event.target.value })} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm text-[var(--color-text-primary)]" />
            </label>
            <label className="grid gap-1 text-xs text-[var(--color-text-muted)]">
              结束时间
              <input type="datetime-local" value={form.endsAt} onChange={(event) => setForm({ ...form, endsAt: event.target.value })} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm text-[var(--color-text-primary)]" />
            </label>
          </div>
          <div className="grid gap-3 md:grid-cols-[180px_1fr]">
            <select value={form.targetType} onChange={(event) => setForm({ ...form, targetType: event.target.value, targetValue: '' })} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm">
              {TARGET_OPTIONS.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
            </select>
            <input value={form.targetValue} onChange={(event) => setForm({ ...form, targetValue: event.target.value })} disabled={form.targetType === 'all'} placeholder={targetHelp} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm disabled:opacity-60" />
          </div>
          <div className="flex flex-wrap gap-4 text-sm text-[var(--color-text-secondary)]">
            <label className="inline-flex items-center gap-2"><input type="checkbox" checked={form.pinned} onChange={(event) => setForm({ ...form, pinned: event.target.checked })} />置顶</label>
            <label className="inline-flex items-center gap-2"><input type="checkbox" checked={form.forcePopup} onChange={(event) => setForm({ ...form, forcePopup: event.target.checked, displayMode: event.target.checked ? 'popup' : form.displayMode })} />强弹窗</label>
          </div>
          <div className="flex flex-wrap gap-2">
            <button type="submit" disabled={loading} className="inline-flex items-center gap-1.5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-4 py-2 text-xs font-medium text-[var(--color-text-secondary)] disabled:opacity-50"><Pencil size={13} />保存草稿</button>
            <button type="button" disabled={loading} onClick={() => selected ? void runAction('publish') : void saveAnnouncement('published')} className="inline-flex items-center gap-1.5 rounded-xl bg-[var(--color-accent)] px-4 py-2 text-xs font-semibold text-white disabled:opacity-50"><Send size={13} />发布</button>
            <button type="button" disabled={loading || !selected} onClick={() => void runAction('schedule')} className="inline-flex items-center gap-1.5 rounded-xl border border-[var(--color-border)] px-4 py-2 text-xs text-[var(--color-text-secondary)] disabled:opacity-50"><CalendarClock size={13} />定时</button>
            <button type="button" disabled={loading || !selected} onClick={() => void runAction('withdraw')} className="inline-flex items-center gap-1.5 rounded-xl border border-[var(--color-border)] px-4 py-2 text-xs text-[var(--color-text-secondary)] disabled:opacity-50"><Undo2 size={13} />撤回</button>
            <button type="button" disabled={loading || !selected} onClick={() => void runAction('archive')} className="inline-flex items-center gap-1.5 rounded-xl border border-[var(--color-border)] px-4 py-2 text-xs text-[var(--color-text-secondary)] disabled:opacity-50"><Archive size={13} />归档</button>
            <button type="button" disabled={loading || !selected || selected?.status !== 'draft'} onClick={() => void runAction('delete')} className="inline-flex items-center gap-1.5 rounded-xl border border-red-500/30 px-4 py-2 text-xs text-red-600 disabled:opacity-50"><Trash2 size={13} />删除草稿</button>
          </div>
          {message && <p className="text-xs text-[var(--color-text-muted)]">{message}</p>}
        </div>
      </form>
    </div>
  )
}

// buildTarget 根据 GM 表单构造公告投放规则。
function buildTarget(type: string, value: string): AnnouncementTarget {
  if (type === 'all') return { type: 'all' }
  if (type === 'level_range') {
    const [min, max] = value.split(/[,，-]/).map((item) => Number(item.trim()))
    return { type, value: { min: Number.isFinite(min) ? min : 0, max: Number.isFinite(max) ? max : 0 } }
  }
  if (type === 'created_at_range') {
    const [from, to] = value.split(/[,，]/).map((item) => item.trim())
    return { type, value: { from, to } }
  }
  return { type, value: value.split(/[,，\s]+/).map((item) => item.trim()).filter(Boolean) }
}

// targetToInput 把后端投放规则转换为表单输入字符串。
function targetToInput(target?: AnnouncementTarget) {
  if (!target || target.type === 'all') return ''
  if (Array.isArray(target.value)) return target.value.join(',')
  if (target.value && typeof target.value === 'object') {
    const record = target.value as Record<string, unknown>
    if ('min' in record || 'max' in record) return `${record.min ?? ''},${record.max ?? ''}`
    if ('from' in record || 'to' in record) return `${record.from ?? ''},${record.to ?? ''}`
  }
  return typeof target.value === 'string' ? target.value : ''
}

// targetValueHelp 返回不同投放类型的输入提示。
function targetValueHelp(type: string) {
  if (type === 'player_ids') return '多个玩家 ID 用逗号分隔'
  if (type === 'account_ids') return '多个账号 ID 用逗号分隔'
  if (type === 'factions') return 'wei,shu,wu'
  if (type === 'level_range') return '暂不开发：等级来源未稳定，先保留字段'
  if (type === 'created_at_range') return '2026-06-01T00:00:00Z,2026-06-30T23:59:59Z'
  return '全体玩家无需填写'
}

// toDatetimeLocal 把后端时间转为 datetime-local 可用值。
function toDatetimeLocal(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
}

// fromDatetimeLocal 把 datetime-local 输入转换为 ISO 时间。
function fromDatetimeLocal(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toISOString()
}

// formatTime 格式化公告时间。
function formatTime(value?: string) {
  if (!value) return '未设置'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
}

// statusLabel 返回公告状态中文名。
function statusLabel(value: string) {
  return STATUS_OPTIONS.find((item) => item.value === value)?.label ?? value
}

// typeLabel 返回公告类型中文名。
function typeLabel(value: string) {
  return TYPE_OPTIONS.find((item) => item.value === value)?.label ?? value
}
