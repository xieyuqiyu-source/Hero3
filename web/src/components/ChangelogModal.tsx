import { useState, useEffect, type FC } from 'react'
import { Sparkles } from 'lucide-react'
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

  // 只显示用户没看过的更新
  const readVersion = getReadVersion()
  const unreadEntries = changelog.filter((e) => e.version > readVersion)
  const entriesToShow = unreadEntries.length > 0 ? unreadEntries : [changelog[0]]

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="更新公告"
      footer={
        <button
          type="button"
          onClick={handleClose}
          className="
            w-full px-4 py-2.5 rounded-xl text-sm font-semibold
            bg-[var(--color-accent)] text-white
            hover:-translate-y-0.5 cursor-pointer
            transition-all duration-200
          "
        >
          知道了
        </button>
      }
    >
      <div className="space-y-5">
        {entriesToShow.map((entry) => (
          <ChangelogSection key={entry.version} entry={entry} />
        ))}
      </div>
    </Modal>
  )
}

const ChangelogSection: FC<{ entry: ChangelogEntry }> = ({ entry }) => (
  <div>
    <div className="flex items-center gap-2 mb-2">
      <Sparkles size={14} className="text-amber-500" />
      <span className="text-sm font-semibold text-[var(--color-text-primary)]">{entry.title}</span>
      <span className="text-[10px] text-[var(--color-text-muted)] ml-auto">{entry.date}</span>
    </div>
    <ul className="space-y-1.5 pl-5">
      {entry.items.map((item, i) => (
        <li
          key={i}
          className="text-xs text-[var(--color-text-secondary)] relative before:content-['•'] before:absolute before:-left-3 before:text-[var(--color-accent)]"
        >
          {item}
        </li>
      ))}
    </ul>
  </div>
)

export default ChangelogModal
