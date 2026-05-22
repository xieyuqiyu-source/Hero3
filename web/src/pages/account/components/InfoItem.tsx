import { type FC } from 'react'

interface InfoItemProps {
  label: string
  value: string
  highlight?: boolean
}

const InfoItem: FC<InfoItemProps> = ({ label, value, highlight }) => (
  <div className="px-3 py-2.5 rounded-xl bg-[var(--color-surface-dim)] border border-[var(--color-border)]">
    <div className="text-[10px] text-[var(--color-text-muted)] mb-0.5">{label}</div>
    <div className={`text-sm font-medium ${highlight ? 'text-[var(--color-accent)]' : 'text-[var(--color-text-primary)]'}`}>
      {value}
    </div>
  </div>
)

export default InfoItem
