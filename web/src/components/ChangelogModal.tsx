import { useState, useEffect, type FC } from 'react'
import { Scroll, Swords, Star } from 'lucide-react'
import { Modal } from '@/components/ui'
import { changelog, LATEST_VERSION, type ChangelogEntry } from '@/data/changelog'

const STORAGE_KEY = 'hero3_changelog_read_version'

function getReadVersion(): number {
  const stored = localStorage.getItem(STORAGE_KEY)
  return stored ? parseInt(stored, 10) : 0
}

function markAsRead() {
  localStorage.setItem(STORAGE_KEY, String(LATEST_VERSION))
}

const ChangelogModal: FC = () => {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    const readVersion = getReadVersion()
    if (readVersion < LATEST_VERSION) {
      setOpen(true)
    }
  }, [])

  const handleClose = () => {
    markAsRead()
    setOpen(false)
  }

  const readVersion = getReadVersion()
  const unreadEntries = changelog.filter((e) => e.version > readVersion)
  const entriesToShow = unreadEntries.length > 0 ? unreadEntries : [changelog[0]]

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title=""
      footer={
        <button
          type="button"
          onClick={handleClose}
          className="
            w-full px-4 py-3 rounded-xl text-sm font-bold tracking-wide
            bg-gradient-to-r from-amber-500 to-orange-500 text-white
            hover:-translate-y-0.5 cursor-pointer
            transition-all duration-200
            shadow-[0_6px_20px_rgba(245,158,11,0.3)]
          "
        >
          朕已阅
        </button>
      }
    >
      <div className="space-y-6">
        {/* Header banner */}
        <div className="relative -mx-5 -mt-4 px-5 pt-6 pb-5 bg-gradient-to-br from-amber-500/10 via-orange-500/5 to-transparent border-b border-amber-500/20">
          <div className="absolute top-3 right-4 opacity-10">
            <Swords size={48} className="text-amber-500" />
          </div>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-amber-500 to-orange-600 flex items-center justify-center shadow-[0_4px_12px_rgba(245,158,11,0.3)]">
              <Scroll size={20} className="text-white" />
            </div>
            <div>
              <h2 className="text-lg font-black text-[var(--color-text-primary)]" style={{ fontFamily: "'STKaiti', 'KaiTi', 'SimKai', serif" }}>
                战报传来
              </h2>
              <p className="text-[11px] text-[var(--color-text-muted)]">英雄三国 · 更新公告</p>
            </div>
          </div>
        </div>

        {/* Entries */}
        {entriesToShow.map((entry) => (
          <ChangelogSection key={entry.version} entry={entry} />
        ))}

        {/* QQ 群 */}
        <div className="relative pt-4 border-t border-[var(--color-border)]">
          <div className="flex items-center justify-center gap-2 px-4 py-3 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
            <span className="text-xs text-[var(--color-text-secondary)]">更多内容交流请加入 QQ 群</span>
            <span className="text-xs font-bold text-[var(--color-accent)]">1101370293</span>
          </div>
        </div>
      </div>
    </Modal>
  )
}

const ChangelogSection: FC<{ entry: ChangelogEntry }> = ({ entry }) => (
  <div className="relative pl-4 border-l-2 border-amber-500/30">
    {/* Version badge */}
    <div className="absolute -left-[7px] top-0 w-3 h-3 rounded-full bg-amber-500 border-2 border-[var(--color-surface)]" />

    <div className="flex items-center gap-2 mb-2.5">
      <span className="text-sm font-bold text-[var(--color-text-primary)]">{entry.title}</span>
      <span className="text-[9px] px-1.5 py-0.5 rounded-md bg-amber-500/15 text-amber-600 font-bold">
        v{entry.version}
      </span>
      <span className="text-[10px] text-[var(--color-text-muted)] ml-auto">{entry.date}</span>
    </div>

    <div className="space-y-2">
      {entry.items.map((item, i) => (
        <div
          key={i}
          className="flex items-start gap-2 px-3 py-2 rounded-lg bg-[var(--color-surface-dim)]/60 border border-[var(--color-border)]"
        >
          <Star size={10} className="text-amber-500 mt-0.5 flex-shrink-0" />
          <span className="text-xs text-[var(--color-text-secondary)] leading-relaxed">{item}</span>
        </div>
      ))}
    </div>
  </div>
)

export default ChangelogModal
