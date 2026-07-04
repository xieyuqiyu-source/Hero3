/* 本文件实现玩家公告中心，支持公告列表、富文本详情、历史筛选和详情阅读。 */
import { useCallback, useEffect, useMemo, useState, type FC } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Megaphone, Pin, Clock, AlertTriangle } from 'lucide-react'
import { gameApi } from '@/api/game'
import { useGameStore } from '@/store/gameStore'
import type { AnnouncementDetail, AnnouncementSummary, AnnouncementType } from '@/types/game'

const PAGE_SIZE = 20

const TYPE_OPTIONS: Array<{ key: string; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 'system', label: '系统' },
  { key: 'maintenance', label: '维护' },
  { key: 'update', label: '更新' },
  { key: 'activity', label: '活动' },
  { key: 'compensation', label: '补偿' },
  { key: 'emergency', label: '紧急' },
  { key: 'history', label: '历史' },
]

const TYPE_LABELS: Record<string, string> = {
  system: '系统',
  maintenance: '维护',
  update: '更新',
  activity: '活动',
  compensation: '补偿',
  emergency: '紧急',
}

const NoticePage: FC = () => {
  const activePlayerId = useGameStore((s) => s.activePlayerId)
  const [searchParams, setSearchParams] = useSearchParams()
  const [activeType, setActiveType] = useState(searchParams.get('type') || 'all')
  const [items, setItems] = useState<AnnouncementSummary[]>([])
  const [selected, setSelected] = useState<AnnouncementDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const includeArchived = activeType === 'history'
  const queryType = activeType === 'all' || activeType === 'history' ? undefined : activeType

  const loadList = useCallback(async (nextPage = page) => {
    if (!activePlayerId) return
    setLoading(true)
    try {
      const result = await gameApi.listAnnouncements(activePlayerId, {
        type: queryType,
        includeArchived,
        page: nextPage,
        pageSize: PAGE_SIZE,
      })
      const nextItems = Array.isArray(result.items) ? result.items : []
      const nextTotal = typeof result.total === 'number' ? result.total : nextItems.length
      setItems(nextItems)
      setTotal(nextTotal)
      if (nextPage > Math.max(1, Math.ceil(nextTotal / PAGE_SIZE))) {
        setPage(1)
      }
    } finally {
      setLoading(false)
    }
  }, [activePlayerId, includeArchived, page, queryType])

  const openDetail = useCallback(async (announcement: AnnouncementSummary) => {
    if (!activePlayerId) return
    setDetailLoading(true)
    try {
      const detail = await gameApi.getAnnouncement(activePlayerId, announcement.id)
      setSelected(detail)
      setSearchParams({ announcementId: announcement.id, type: activeType })
      if (!detail.isRead) {
        await gameApi.markAnnouncementRead(activePlayerId, announcement.id)
        setItems((prev) => prev.map((item) => item.id === announcement.id ? { ...item, isRead: true } : item))
        setSelected({ ...detail, isRead: true })
      }
    } finally {
      setDetailLoading(false)
    }
  }, [activePlayerId, activeType, setSearchParams])

  useEffect(() => {
    void loadList(1)
  }, [activePlayerId, activeType])

  useEffect(() => {
    void loadList(page)
  }, [page])

  useEffect(() => {
    const announcementId = searchParams.get('announcementId')
    if (!announcementId || !activePlayerId) return
    const cached = items.find((item) => item.id === announcementId)
    if (cached) {
      void openDetail(cached)
    }
  }, [activePlayerId, items, openDetail, searchParams])

  const title = useMemo(() => selected?.title || '公告中心', [selected])

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-[var(--color-text-primary)]">{title}</h1>
          <p className="mt-1 text-sm text-[var(--color-text-secondary)]">公告只承载运营说明，补偿和奖励请前往信函查看。</p>
        </div>
        {selected && (
          <button
            type="button"
            onClick={() => {
              setSelected(null)
              setSearchParams(activeType === 'all' ? {} : { type: activeType })
            }}
            className="rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm text-[var(--color-text-secondary)]"
          >
            返回列表
          </button>
        )}
      </div>

      {!selected && (
        <div className="flex flex-wrap gap-2">
          {TYPE_OPTIONS.map((item) => (
            <button
              key={item.key}
              type="button"
              onClick={() => {
                setActiveType(item.key)
                setPage(1)
                setSearchParams(item.key === 'all' ? {} : { type: item.key })
              }}
              className={`rounded-lg border px-3 py-2 text-sm transition-colors ${activeType === item.key ? 'border-[var(--color-accent-border)] bg-[var(--color-accent-light)] text-[var(--color-accent)]' : 'border-[var(--color-border)] text-[var(--color-text-secondary)]'}`}
            >
              {item.label}
            </button>
          ))}
        </div>
      )}

      {selected ? (
        <article className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-5">
          {detailLoading && <p className="text-sm text-[var(--color-text-muted)]">公告详情加载中...</p>}
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <NoticeTypeBadge type={selected.type} />
            {selected.pinned && <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-1 text-xs text-amber-600"><Pin size={12} />置顶</span>}
            {selected.forcePopup && <span className="inline-flex items-center gap-1 rounded-full bg-red-500/10 px-2 py-1 text-xs text-red-600"><AlertTriangle size={12} />强弹窗</span>}
          </div>
          <h2 className="text-2xl font-bold text-[var(--color-text-primary)]">{selected.title}</h2>
          <p className="mt-2 text-sm text-[var(--color-text-muted)]">{formatTime(selected.publishedAt)}{selected.endsAt ? ` · 截止 ${formatTime(selected.endsAt)}` : ''}</p>
          <AnnouncementContent content={selected.content} />
        </article>
      ) : (
        <div className="space-y-3">
          {loading && <p className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-8 text-center text-sm text-[var(--color-text-muted)]">公告加载中...</p>}
          {!loading && items.length === 0 && <p className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-8 text-center text-sm text-[var(--color-text-muted)]">暂无公告</p>}
          {items.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => void openDetail(item)}
              className="w-full rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 text-left transition-all hover:border-[var(--color-accent-border)] hover:shadow-sm"
            >
              <div className="flex flex-wrap items-center gap-2">
                <NoticeTypeBadge type={item.type} />
                {item.pinned && <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-1 text-xs text-amber-600"><Pin size={12} />置顶</span>}
                {!item.isRead && <span className="rounded-full bg-red-500 px-1.5 py-0.5 text-[10px] font-semibold text-white">未读</span>}
              </div>
              <h2 className="mt-3 text-base font-bold text-[var(--color-text-primary)]">{item.title}</h2>
              <p className="mt-1 line-clamp-2 text-sm leading-6 text-[var(--color-text-secondary)]">{item.summary}</p>
              <p className="mt-3 inline-flex items-center gap-1 text-xs text-[var(--color-text-muted)]"><Clock size={12} />{formatTime(item.publishedAt)}</p>
            </button>
          ))}
          {totalPages > 1 && (
            <div className="flex items-center justify-end gap-2">
              <button type="button" disabled={page <= 1} onClick={() => setPage((v) => Math.max(1, v - 1))} className="rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm disabled:opacity-50">上一页</button>
              <span className="text-sm text-[var(--color-text-muted)]">{page}/{totalPages}</span>
              <button type="button" disabled={page >= totalPages} onClick={() => setPage((v) => Math.min(totalPages, v + 1))} className="rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm disabled:opacity-50">下一页</button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

const NoticeTypeBadge: FC<{ type: AnnouncementType | string }> = ({ type }) => (
  <span className="inline-flex items-center gap-1 rounded-full bg-[var(--color-accent-light)] px-2 py-1 text-xs font-semibold text-[var(--color-accent)]">
    <Megaphone size={12} />
    {TYPE_LABELS[type] ?? '公告'}
  </span>
)

const HTML_ANNOUNCEMENT_TAGS = new Set(['ARTICLE', 'SECTION', 'H3', 'P', 'UL', 'OL', 'LI', 'STRONG', 'EM', 'BR'])

// AnnouncementContent 渲染公告正文，自动公告使用受限 HTML，普通公告按纯文本换行展示。
const AnnouncementContent: FC<{ content: string }> = ({ content }) => {
  const sanitizedHtml = useMemo(() => sanitizeAnnouncementHtml(content), [content])
  if (sanitizedHtml) {
    return (
      <div
        className="mt-5 text-sm leading-7 text-[var(--color-text-secondary)] [&_h3]:mb-2 [&_h3]:mt-4 [&_h3]:text-base [&_h3]:font-bold [&_h3]:text-[var(--color-text-primary)] [&_li]:mb-1.5 [&_p]:mb-3 [&_ul]:list-disc [&_ul]:space-y-1.5 [&_ul]:pl-5"
        dangerouslySetInnerHTML={{ __html: sanitizedHtml }}
      />
    )
  }
  return <div className="mt-5 whitespace-pre-wrap text-sm leading-7 text-[var(--color-text-secondary)]">{content}</div>
}

// sanitizeAnnouncementHtml 只允许公告富文本需要的少量标签和安全 class。
function sanitizeAnnouncementHtml(content: string) {
  const trimmed = content.trim()
  if (!/^<(article|section|p|ul|ol|li|h3|strong|em|br)(\s|>)/i.test(trimmed)) {
    return ''
  }
  const parser = new DOMParser()
  const document = parser.parseFromString(trimmed, 'text/html')
  const sanitizeNode = (node: Node): Node | null => {
    if (node.nodeType === Node.TEXT_NODE) {
      return document.createTextNode(node.textContent ?? '')
    }
    if (node.nodeType !== Node.ELEMENT_NODE) {
      return null
    }
    const element = node as HTMLElement
    if (!HTML_ANNOUNCEMENT_TAGS.has(element.tagName)) {
      const fragment = document.createDocumentFragment()
      element.childNodes.forEach((child) => {
        const sanitized = sanitizeNode(child)
        if (sanitized) fragment.appendChild(sanitized)
      })
      return fragment
    }
    const next = document.createElement(element.tagName.toLowerCase())
    const className = element.getAttribute('class') ?? ''
    if (className.split(/\s+/).every((item) => item === '' || item.startsWith('hero3-announcement'))) {
      next.setAttribute('class', className)
    }
    element.childNodes.forEach((child) => {
      const sanitized = sanitizeNode(child)
      if (sanitized) next.appendChild(sanitized)
    })
    return next
  }
  const fragment = document.createDocumentFragment()
  document.body.childNodes.forEach((child) => {
    const sanitized = sanitizeNode(child)
    if (sanitized) fragment.appendChild(sanitized)
  })
  const wrapper = document.createElement('div')
  wrapper.appendChild(fragment)
  return wrapper.innerHTML
}

function formatTime(value?: string) {
  if (!value) return '未发布'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

export default NoticePage
