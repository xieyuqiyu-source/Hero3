/* GM 公告管理页面，负责公告列表、编辑、发布和下架。 */

import { useEffect, useState, type FormEvent } from 'react'
import { Archive, Edit3, Loader2, Megaphone, Pin, Plus, RefreshCw, Send, Trash2, X } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { Announcement, AnnouncementInput, AnnouncementType } from '@/types'

const TYPE_OPTIONS: Array<{ value: AnnouncementType; label: string }> = [
  { value: 'system', label: '系统公告' },
  { value: 'maintenance', label: '维护公告' },
  { value: 'event', label: '活动公告' },
  { value: 'update', label: '更新公告' },
]

const STATUS_LABELS: Record<string, string> = {
  draft: '草稿',
  published: '已发布',
  archived: '已下架',
}

const EMPTY_FORM: AnnouncementInput = {
  title: '',
  content: '',
  type: 'system',
  pinned: false,
  priority: 0,
  startsAt: '',
  endsAt: '',
}

// toDateTimeLocal 将 ISO 时间转成 datetime-local 可用值。
function toDateTimeLocal(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const offsetMs = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16)
}

// toISOStringOrEmpty 将 datetime-local 值转成后端统一时间。
function toISOStringOrEmpty(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toISOString()
}

// formatDateTime 格式化公告时间。
function formatDateTime(value?: string) {
  if (!value) return '未设置'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN')
}

// buildFormFromAnnouncement 将公告数据填入表单。
function buildFormFromAnnouncement(announcement: Announcement): AnnouncementInput {
  return {
    title: announcement.title,
    content: announcement.content,
    type: announcement.type,
    pinned: announcement.pinned,
    priority: announcement.priority,
    startsAt: toDateTimeLocal(announcement.startsAt),
    endsAt: toDateTimeLocal(announcement.endsAt),
  }
}

export default function AnnouncementAdminPanel() {
  const [announcements, setAnnouncements] = useState<Announcement[]>([])
  const [form, setForm] = useState<AnnouncementInput>(EMPTY_FORM)
  const [editingId, setEditingId] = useState('')
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  // loadAnnouncements 加载 GM 公告列表。
  const loadAnnouncements = async (options: { silent?: boolean } = {}) => {
    if (!options.silent) setLoading(true)
    setMessage('')
    try {
      const result = await adminApi.getAnnouncements()
      setAnnouncements(result.announcements)
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '公告加载失败')
    } finally {
      if (!options.silent) setLoading(false)
    }
  }

  useEffect(() => {
    void loadAnnouncements()
  }, [])

  // validateForm 校验公告表单。
  const validateForm = () => {
    if (!form.title.trim()) return '标题必填'
    if (!form.content.trim()) return '正文必填'
    if (form.startsAt && form.endsAt && new Date(form.startsAt).getTime() > new Date(form.endsAt).getTime()) {
      return '开始时间不能晚于结束时间'
    }
    return ''
  }

  // buildPayload 构造提交给后端的公告输入。
  const buildPayload = (): AnnouncementInput => ({
    title: form.title.trim(),
    content: form.content.trim(),
    type: form.type,
    pinned: form.pinned,
    priority: Number(form.priority) || 0,
    startsAt: toISOStringOrEmpty(form.startsAt),
    endsAt: toISOStringOrEmpty(form.endsAt),
  })

  // resetForm 清空编辑态。
  const resetForm = () => {
    setForm(EMPTY_FORM)
    setEditingId('')
  }

  // handleSubmit 新建或更新公告。
  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    const validationMessage = validateForm()
    if (validationMessage) {
      setMessage(validationMessage)
      return
    }
    setSubmitting(true)
    setMessage('')
    try {
      if (editingId) {
        await adminApi.updateAnnouncement(editingId, buildPayload())
        setMessage('公告已更新')
      } else {
        await adminApi.createAnnouncement(buildPayload())
        setMessage('公告草稿已创建')
      }
      resetForm()
      await loadAnnouncements({ silent: true })
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '保存失败')
    } finally {
      setSubmitting(false)
    }
  }

  // handleAction 执行发布、下架或删除。
  const handleAction = async (announcement: Announcement, action: 'publish' | 'archive' | 'delete') => {
    const actionLabel = action === 'publish' ? '发布' : action === 'archive' ? '下架' : '删除'
    if (!window.confirm(`确认${actionLabel}公告「${announcement.title}」？`)) return
    setLoading(true)
    setMessage('')
    try {
      if (action === 'publish') await adminApi.publishAnnouncement(announcement.id)
      if (action === 'archive') await adminApi.archiveAnnouncement(announcement.id)
      if (action === 'delete') await adminApi.deleteAnnouncement(announcement.id)
      setMessage(`公告已${actionLabel}`)
      await loadAnnouncements({ silent: true })
    } catch (error) {
      setMessage(error instanceof Error ? error.message : `${actionLabel}失败`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="grid gap-4">
      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-3 border-b border-[var(--color-border)]">
          <Megaphone size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">公告管理</h2>
          <button
            type="button"
            onClick={() => void loadAnnouncements()}
            disabled={loading}
            className="ml-auto inline-flex h-8 w-8 items-center justify-center rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] disabled:opacity-50 cursor-pointer transition-colors"
            aria-label="刷新公告"
          >
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="grid gap-3 p-4">
          <div className="grid gap-3 md:grid-cols-[1fr_180px_120px]">
            <input
              value={form.title}
              onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))}
              maxLength={120}
              placeholder="公告标题"
              className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
            />
            <select
              value={form.type}
              onChange={(event) => setForm((current) => ({ ...current, type: event.target.value as AnnouncementType }))}
              className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
            >
              {TYPE_OPTIONS.map((item) => (
                <option key={item.value} value={item.value}>{item.label}</option>
              ))}
            </select>
            <input
              type="number"
              value={form.priority}
              onChange={(event) => setForm((current) => ({ ...current, priority: Number(event.target.value) }))}
              placeholder="优先级"
              className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
            />
          </div>

          <textarea
            value={form.content}
            onChange={(event) => setForm((current) => ({ ...current, content: event.target.value }))}
            maxLength={10000}
            rows={6}
            placeholder="公告正文"
            className="resize-y rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent-border)]"
          />

          <div className="grid gap-3 md:grid-cols-[1fr_1fr_120px]">
            <label className="grid gap-1 text-xs text-[var(--color-text-muted)]">
              开始时间
              <input
                type="datetime-local"
                value={form.startsAt}
                onChange={(event) => setForm((current) => ({ ...current, startsAt: event.target.value }))}
                className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
              />
            </label>
            <label className="grid gap-1 text-xs text-[var(--color-text-muted)]">
              结束时间
              <input
                type="datetime-local"
                value={form.endsAt}
                onChange={(event) => setForm((current) => ({ ...current, endsAt: event.target.value }))}
                className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
              />
            </label>
            <label className="flex items-center gap-2 self-end rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2 text-sm text-[var(--color-text-secondary)]">
              <input
                type="checkbox"
                checked={form.pinned}
                onChange={(event) => setForm((current) => ({ ...current, pinned: event.target.checked }))}
                className="h-4 w-4 accent-[var(--color-accent)]"
              />
              置顶
            </label>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <button
              type="submit"
              disabled={submitting}
              className="inline-flex items-center gap-1.5 rounded-xl bg-[var(--color-accent)] px-4 py-2 text-xs font-semibold text-white hover:opacity-90 disabled:opacity-50 cursor-pointer transition-opacity"
            >
              {submitting ? <Loader2 size={13} className="animate-spin" /> : <Plus size={13} />}
              {editingId ? '保存公告' : '新建草稿'}
            </button>
            {editingId && (
              <button
                type="button"
                onClick={resetForm}
                className="inline-flex items-center gap-1.5 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-4 py-2 text-xs font-medium text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] cursor-pointer transition-colors"
              >
                <X size={13} />
                取消编辑
              </button>
            )}
            {message && <span className="text-xs text-[var(--color-text-muted)]">{message}</span>}
          </div>
        </form>
      </section>

      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
        <div className="flex items-center gap-2 px-4 py-3 border-b border-[var(--color-border)]">
          <span className="text-sm font-semibold text-[var(--color-text-primary)]">公告列表</span>
          <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">共 {announcements.length} 条</span>
        </div>

        <div className="space-y-2 p-4">
          {loading && announcements.length === 0 ? (
            <div className="flex items-center justify-center py-8 text-sm text-[var(--color-text-muted)]">
              <Loader2 size={16} className="mr-2 animate-spin text-[var(--color-accent)]" />
              公告加载中...
            </div>
          ) : announcements.length === 0 ? (
            <div className="py-8 text-center text-sm text-[var(--color-text-muted)]">暂无公告</div>
          ) : announcements.map((announcement) => (
            <div key={announcement.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-3">
              <div className="flex flex-wrap items-start gap-2">
                <div className="min-w-[220px] flex-1">
                  <div className="flex flex-wrap items-center gap-1.5">
                    {announcement.pinned && (
                      <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-0.5 text-[10px] font-semibold text-amber-600">
                        <Pin size={10} />
                        置顶
                      </span>
                    )}
                    <span className="rounded-full bg-[var(--color-surface)] px-2 py-0.5 text-[10px] text-[var(--color-text-secondary)]">
                      {TYPE_OPTIONS.find((item) => item.value === announcement.type)?.label ?? '系统公告'}
                    </span>
                    <span className="rounded-full bg-[var(--color-surface)] px-2 py-0.5 text-[10px] text-[var(--color-text-secondary)]">
                      {STATUS_LABELS[announcement.status] ?? announcement.status}
                    </span>
                    <span className="text-[10px] text-[var(--color-text-muted)]">P{announcement.priority}</span>
                  </div>
                  <h3 className="mt-2 text-sm font-semibold text-[var(--color-text-primary)]">{announcement.title}</h3>
                  <p className="mt-1 line-clamp-2 text-xs leading-5 text-[var(--color-text-secondary)]">{announcement.content}</p>
                  <div className="mt-2 flex flex-wrap gap-3 text-[10px] text-[var(--color-text-muted)]">
                    <span>开始：{formatDateTime(announcement.startsAt)}</span>
                    <span>结束：{formatDateTime(announcement.endsAt)}</span>
                    <span>更新：{formatDateTime(announcement.updatedAt)}</span>
                  </div>
                </div>

                <div className="flex shrink-0 flex-wrap gap-1.5">
                  <button
                    type="button"
                    onClick={() => {
                      setEditingId(announcement.id)
                      setForm(buildFormFromAnnouncement(announcement))
                    }}
                    className="inline-flex h-8 items-center gap-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 text-[11px] text-[var(--color-text-secondary)] hover:border-[var(--color-accent-border)] hover:text-[var(--color-accent)] cursor-pointer transition-colors"
                  >
                    <Edit3 size={12} />
                    编辑
                  </button>
                  <button
                    type="button"
                    onClick={() => void handleAction(announcement, 'publish')}
                    disabled={announcement.status === 'published'}
                    className="inline-flex h-8 items-center gap-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 text-[11px] text-[var(--color-text-secondary)] hover:border-emerald-400 hover:text-emerald-600 disabled:opacity-40 cursor-pointer transition-colors"
                  >
                    <Send size={12} />
                    发布
                  </button>
                  <button
                    type="button"
                    onClick={() => void handleAction(announcement, 'archive')}
                    disabled={announcement.status === 'archived'}
                    className="inline-flex h-8 items-center gap-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 text-[11px] text-[var(--color-text-secondary)] hover:border-amber-400 hover:text-amber-600 disabled:opacity-40 cursor-pointer transition-colors"
                  >
                    <Archive size={12} />
                    下架
                  </button>
                  <button
                    type="button"
                    onClick={() => void handleAction(announcement, 'delete')}
                    className="inline-flex h-8 items-center gap-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2.5 text-[11px] text-[var(--color-text-secondary)] hover:border-red-400 hover:text-red-500 cursor-pointer transition-colors"
                  >
                    <Trash2 size={12} />
                    删除
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
