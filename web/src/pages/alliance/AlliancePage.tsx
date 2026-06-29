// 联盟页面占位，等待后续联盟系统接入。
import { type FC } from 'react'
import { UsersRound } from 'lucide-react'

/** 渲染联盟占位页。 */
const AlliancePage: FC = () => (
  <section className="space-y-4">
    <div className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-5 shadow-[0_10px_30px_rgba(15,23,42,0.06)]">
      <div className="flex items-center gap-3">
        <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-[var(--color-accent-light)] text-[var(--color-accent)]">
          <UsersRound size={22} />
        </div>
        <div>
          <h1 className="text-xl font-bold text-[var(--color-text-primary)]">联盟</h1>
          <p className="text-sm text-[var(--color-text-secondary)]">联盟系统暂未开放。</p>
        </div>
      </div>
    </div>
  </section>
)

export default AlliancePage
