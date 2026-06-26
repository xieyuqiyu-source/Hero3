/* 帮助中心页面提供 Wiki 风格的项目文档查阅入口。 */

import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { BookOpen, ChevronDown, FileText, FolderOpen, RefreshCw, Search } from 'lucide-react'
import { gameApi, type HelpDocument, type HelpDocumentSummary } from '@/api/game'

interface HelpCategory {
  key: string
  label: string
  description: string
  documents: HelpDocumentSummary[]
}

interface HeadingAnchor {
  id: string
  level: number
  text: string
}

const CATEGORY_META: Record<string, { label: string; description: string }> = {
  '01-project': { label: '项目总览', description: '项目定位、最终目标和当前方向' },
  '02-core': { label: '核心系统', description: '核心边界、基础循环、核心管线' },
  '03-registries': { label: '注册表', description: '资源、建筑、兵种、奖励、加成等注册表' },
  '04-module-development': { label: '模块开发', description: '新增功能、建筑、兵种、副本、据点的接入方式' },
  '05-gameplay-planning': { label: '玩法规划', description: '未来副本、据点、NPC 机制和功能规划' },
  '06-database': { label: '数据库', description: '权威表、state_json、迁移规则' },
  '07-api': { label: 'API 接口', description: '接口模块、响应、错误和帮助文档 API' },
  '08-directory': { label: '目录结构', description: '项目目录、Go 分层、前端目录' },
  '09-development-rules': { label: '开发规则', description: '项目决策规则和文档维护规则' },
  '10-memory': { label: '项目记忆', description: 'memory 目录和会话延续规则' },
  uncategorized: { label: '未分类', description: '尚未归类的帮助文档' },
}

/** 帮助页面主组件，负责加载文档索引和当前文档。 */
function HelpPage() {
  const [documents, setDocuments] = useState<HelpDocumentSummary[]>([])
  const [activeId, setActiveId] = useState('')
  const [activeDocument, setActiveDocument] = useState<HelpDocument | null>(null)
  const [query, setQuery] = useState('')
  const [openCategories, setOpenCategories] = useState<Record<string, boolean>>({})
  const [loading, setLoading] = useState(false)
  const [documentLoading, setDocumentLoading] = useState(false)
  const [error, setError] = useState('')

  const categories = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    const grouped = new Map<string, HelpDocumentSummary[]>()

    documents.forEach((document) => {
      const matches = !keyword
        || document.title.toLowerCase().includes(keyword)
        || document.id.toLowerCase().includes(keyword)
        || document.excerpt.toLowerCase().includes(keyword)
      if (!matches) return

      const categoryKey = document.id.includes('/') ? document.id.split('/')[0] : 'uncategorized'
      const categoryDocuments = grouped.get(categoryKey) ?? []
      categoryDocuments.push(document)
      grouped.set(categoryKey, categoryDocuments)
    })

    return Array.from(grouped.entries())
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, categoryDocuments]) => ({
        key,
        label: CATEGORY_META[key]?.label ?? key,
        description: CATEGORY_META[key]?.description ?? '自定义帮助文档',
        documents: categoryDocuments.sort((left, right) => left.id.localeCompare(right.id)),
      }))
  }, [documents, query])

  const headings = useMemo(() => extractHeadings(activeDocument?.content ?? ''), [activeDocument])

  /** loadDocuments 拉取帮助文档索引并设置默认选中文档。 */
  const loadDocuments = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const response = await gameApi.listHelpDocuments()
      setDocuments(response.documents)
      setOpenCategories(defaultOpenCategories(response.documents))
      setActiveId((current) => current || defaultActiveDocument(response.documents))
    } catch {
      setError('帮助文档加载失败')
    } finally {
      setLoading(false)
    }
  }, [])

  /** loadDocument 拉取当前选中文档正文。 */
  const loadDocument = useCallback(async (documentId: string) => {
    setDocumentLoading(true)
    setError('')
    try {
      const response = await gameApi.getHelpDocument(documentId)
      setActiveDocument(response.document)
    } catch {
      setError('帮助文档正文加载失败')
    } finally {
      setDocumentLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadDocuments()
  }, [loadDocuments])

  useEffect(() => {
    if (!activeId) return
    void loadDocument(activeId)
  }, [activeId, loadDocument])

  return (
    <section className="space-y-4 help-page">
      <div className="flex flex-col gap-3 rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-[0_10px_30px_rgba(15,23,42,0.06)] lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-[var(--color-accent-light)] text-[var(--color-accent)]">
            <BookOpen size={22} />
          </div>
          <div>
            <h1 className="text-xl font-bold text-[var(--color-text-primary)]">帮助中心</h1>
            <p className="text-sm text-[var(--color-text-secondary)]">项目 Wiki、核心架构、模块开发、数据库和规则查询</p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => void loadDocuments()}
          className="inline-flex h-10 items-center justify-center gap-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 text-sm font-semibold text-[var(--color-text-secondary)] transition hover:border-[var(--color-accent-border)] hover:text-[var(--color-accent)]"
        >
          <RefreshCw size={16} className={loading ? 'animate-spin' : ''} />
          刷新
        </button>
      </div>

      {error && (
        <div className="rounded-xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-600">
          {error}
        </div>
      )}

      <div className="grid gap-4 xl:grid-cols-[320px_minmax(0,1fr)_220px] lg:grid-cols-[300px_minmax(0,1fr)]">
        <aside className="help-sidebar rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3 shadow-[0_10px_30px_rgba(15,23,42,0.05)]">
          <label className="mb-3 flex h-10 items-center gap-2 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3">
            <Search size={16} className="text-[var(--color-text-muted)]" />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="搜索文档"
              className="min-w-0 flex-1 bg-transparent text-sm text-[var(--color-text-primary)] outline-none placeholder:text-[var(--color-text-muted)]"
            />
          </label>

          <div className="space-y-2">
            {categories.map((category) => (
              <HelpCategoryTree
                key={category.key}
                category={category}
                activeId={activeId}
                open={openCategories[category.key] ?? false}
                onToggle={() => setOpenCategories((current) => ({
                  ...current,
                  [category.key]: !(current[category.key] ?? false),
                }))}
                onSelect={setActiveId}
              />
            ))}
            {!loading && categories.length === 0 && (
              <div className="rounded-xl border border-dashed border-[var(--color-border)] px-3 py-8 text-center text-sm text-[var(--color-text-muted)]">
                没有匹配的文档
              </div>
            )}
          </div>
        </aside>

        <article className="min-h-[640px] rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-5 shadow-[0_10px_30px_rgba(15,23,42,0.05)] lg:p-7">
          {documentLoading && (
            <div className="flex min-h-[420px] items-center justify-center text-sm text-[var(--color-text-secondary)]">
              正在读取文档...
            </div>
          )}
          {!documentLoading && activeDocument && (
            <MarkdownDocument content={activeDocument.content} />
          )}
          {!documentLoading && !activeDocument && (
            <div className="flex min-h-[420px] items-center justify-center text-sm text-[var(--color-text-secondary)]">
              请选择左侧文档
            </div>
          )}
        </article>

        <aside className="help-toc hidden rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3 shadow-[0_10px_30px_rgba(15,23,42,0.05)] xl:block">
          <div className="mb-2 px-2 text-xs font-bold tracking-widest text-[var(--color-text-muted)]">文内目录</div>
          <div className="space-y-1">
            {headings.map((heading) => (
              <a
                key={heading.id}
                href={`#${heading.id}`}
                className={`block rounded-lg px-2 py-1.5 text-xs leading-5 text-[var(--color-text-secondary)] hover:bg-[var(--color-accent-light)] hover:text-[var(--color-accent)] ${heading.level === 3 ? 'ml-3' : ''}`}
              >
                {heading.text}
              </a>
            ))}
            {headings.length === 0 && (
              <span className="block px-2 py-2 text-xs text-[var(--color-text-muted)]">暂无目录</span>
            )}
          </div>
        </aside>
      </div>
    </section>
  )
}

/** HelpCategoryTree 渲染左侧分类目录树。 */
function HelpCategoryTree({
  category,
  activeId,
  open,
  onToggle,
  onSelect,
}: {
  category: HelpCategory
  activeId: string
  open: boolean
  onToggle: () => void
  onSelect: (id: string) => void
}) {
  return (
    <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)]">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-start gap-2 px-3 py-2.5 text-left"
      >
        <FolderOpen size={16} className="mt-0.5 text-[var(--color-accent)]" />
        <span className="min-w-0 flex-1">
          <span className="block text-sm font-bold text-[var(--color-text-primary)]">{category.label}</span>
          <span className="mt-0.5 block text-[11px] leading-4 text-[var(--color-text-muted)]">{category.description}</span>
        </span>
        <ChevronDown size={15} className={`mt-0.5 text-[var(--color-text-muted)] transition ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && (
        <div className="border-t border-[var(--color-border)] p-1.5">
          {category.documents.map((document) => (
            <button
              key={document.id}
              type="button"
              onClick={() => onSelect(document.id)}
              className={`flex w-full items-start gap-2 rounded-lg px-2 py-2 text-left transition ${
                activeId === document.id
                  ? 'bg-[var(--color-accent-light)] text-[var(--color-accent)]'
                  : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface)] hover:text-[var(--color-text-primary)]'
              }`}
            >
              <FileText size={14} className="mt-0.5 flex-shrink-0" />
              <span className="min-w-0">
                <span className="block truncate text-xs font-bold">{document.title}</span>
                <span className="mt-0.5 line-clamp-2 block text-[11px] leading-4 opacity-75">{document.excerpt}</span>
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

/** MarkdownDocument 渲染帮助站支持的轻量 Markdown 语法。 */
function MarkdownDocument({ content }: { content: string }) {
  const blocks = useMemo(() => parseMarkdown(content), [content])
  return (
    <div className="help-markdown">
      {blocks}
    </div>
  )
}

/** parseMarkdown 把 Markdown 文本转换成 React 节点。 */
function parseMarkdown(content: string): ReactNode[] {
  const lines = content.replace(/\r\n/g, '\n').split('\n')
  const nodes: ReactNode[] = []
  let listItems: string[] = []
  let orderedListItems: string[] = []
  let paragraph: string[] = []
  let codeLines: string[] = []
  let tableRows: string[][] = []
  let inCodeBlock = false

  /** flushParagraph 输出当前段落。 */
  const flushParagraph = () => {
    if (paragraph.length === 0) return
    nodes.push(<p key={`p-${nodes.length}`}>{paragraph.join(' ')}</p>)
    paragraph = []
  }

  /** flushList 输出当前无序列表。 */
  const flushList = () => {
    if (listItems.length === 0) return
    nodes.push(
      <ul key={`ul-${nodes.length}`}>
        {listItems.map((item) => <li key={item}>{item}</li>)}
      </ul>,
    )
    listItems = []
  }

  /** flushOrderedList 输出当前有序列表。 */
  const flushOrderedList = () => {
    if (orderedListItems.length === 0) return
    nodes.push(
      <ol key={`ol-${nodes.length}`}>
        {orderedListItems.map((item) => <li key={item}>{item}</li>)}
      </ol>,
    )
    orderedListItems = []
  }

  /** flushTable 输出当前表格。 */
  const flushTable = () => {
    if (tableRows.length === 0) return
    const [head, separator, ...body] = tableRows
    const hasSeparator = separator?.every((cell) => /^:?-{3,}:?$/.test(cell.trim())) ?? false
    const bodyRows = hasSeparator ? body : tableRows.slice(1)
    nodes.push(
      <div key={`table-${nodes.length}`} className="help-table-wrap">
        <table>
          <thead>
            <tr>{head.map((cell, index) => <th key={`${cell}-${index}`}>{cell}</th>)}</tr>
          </thead>
          <tbody>
            {bodyRows.map((row, index) => (
              <tr key={`${row.join('-')}-${index}`}>{row.map((cell, cellIndex) => <td key={`${cell}-${cellIndex}`}>{cell}</td>)}</tr>
            ))}
          </tbody>
        </table>
      </div>,
    )
    tableRows = []
  }

  /** flushCode 输出当前代码块。 */
  const flushCode = () => {
    if (codeLines.length === 0) return
    nodes.push(<pre key={`pre-${nodes.length}`}><code>{codeLines.join('\n')}</code></pre>)
    codeLines = []
  }

  lines.forEach((line) => {
    const trimmed = line.trim()
    if (trimmed.startsWith('```')) {
      if (inCodeBlock) {
        flushCode()
        inCodeBlock = false
      } else {
        flushParagraph()
        flushList()
        flushOrderedList()
        flushTable()
        inCodeBlock = true
      }
      return
    }

    if (inCodeBlock) {
      codeLines.push(line)
      return
    }

    if (!trimmed) {
      flushParagraph()
      flushList()
      flushOrderedList()
      flushTable()
      return
    }

    if (trimmed.startsWith('### ')) {
      flushParagraph()
      flushList()
      flushOrderedList()
      flushTable()
      const text = trimmed.slice(4)
      nodes.push(<h3 id={headingID(text)} key={`h3-${nodes.length}`}>{text}</h3>)
      return
    }
    if (trimmed.startsWith('## ')) {
      flushParagraph()
      flushList()
      flushOrderedList()
      flushTable()
      const text = trimmed.slice(3)
      nodes.push(<h2 id={headingID(text)} key={`h2-${nodes.length}`}>{text}</h2>)
      return
    }
    if (trimmed.startsWith('# ')) {
      flushParagraph()
      flushList()
      flushOrderedList()
      flushTable()
      const text = trimmed.slice(2)
      nodes.push(<h1 id={headingID(text)} key={`h1-${nodes.length}`}>{text}</h1>)
      return
    }
    if (trimmed.startsWith('- ')) {
      flushParagraph()
      flushOrderedList()
      flushTable()
      listItems.push(trimmed.slice(2))
      return
    }
    if (/^\d+\.\s+/.test(trimmed)) {
      flushParagraph()
      flushList()
      flushTable()
      orderedListItems.push(trimmed.replace(/^\d+\.\s+/, ''))
      return
    }
    if (trimmed.startsWith('> ')) {
      flushParagraph()
      flushList()
      flushOrderedList()
      flushTable()
      nodes.push(<blockquote key={`quote-${nodes.length}`}>{trimmed.slice(2)}</blockquote>)
      return
    }
    if (trimmed.startsWith('|') && trimmed.endsWith('|')) {
      flushParagraph()
      flushList()
      flushOrderedList()
      tableRows.push(trimmed.slice(1, -1).split('|').map((cell) => cell.trim()))
      return
    }
    flushTable()
    paragraph.push(trimmed)
  })

  flushParagraph()
  flushList()
  flushOrderedList()
  flushTable()
  flushCode()
  return nodes
}

/** defaultOpenCategories 默认展开所有分类，方便第一次浏览。 */
function defaultOpenCategories(documents: HelpDocumentSummary[]) {
  return documents.reduce<Record<string, boolean>>((result, document) => {
    const categoryKey = document.id.includes('/') ? document.id.split('/')[0] : 'uncategorized'
    result[categoryKey] = true
    return result
  }, {})
}

/** defaultActiveDocument 默认打开项目总览。 */
function defaultActiveDocument(documents: HelpDocumentSummary[]) {
  return documents.find((document) => document.id === '01-project/overview')?.id ?? documents[0]?.id ?? ''
}

/** extractHeadings 提取文内二三级标题目录。 */
function extractHeadings(content: string): HeadingAnchor[] {
  return content
    .replace(/\r\n/g, '\n')
    .split('\n')
    .filter((line) => line.startsWith('## ') || line.startsWith('### '))
    .map((line) => {
      const level = line.startsWith('### ') ? 3 : 2
      const text = line.slice(level + 1).trim()
      return { id: headingID(text), level, text }
    })
}

/** headingID 生成稳定标题锚点。 */
function headingID(text: string) {
  return encodeURIComponent(text.trim().replace(/\s+/g, '-'))
}

export default HelpPage
