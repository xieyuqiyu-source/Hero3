/* 本文件展示 GM 后续配置副本背景图所需的上传参数。 */
import { Image, UploadCloud } from 'lucide-react'

const DUNGEON_IMAGE_CONFIGS = [
  {
    slotKey: 'reincarnation-abyss',
    name: '轮回绝境',
    category: '常驻副本',
    currentAsset: 'web/src/assets/dungeons/reincarnation-abyss.webp',
  },
  {
    slotKey: 'kings-war',
    name: '万王争霸',
    category: '常驻副本',
    currentAsset: 'web/src/assets/dungeons/kings-war.webp',
  },
  {
    slotKey: 'famous-generals',
    name: '天下名将',
    category: '常驻副本',
    currentAsset: 'web/src/assets/dungeons/famous-generals.webp',
  },
  {
    slotKey: 'god-demon-battlefield',
    name: '神魔战场',
    category: '限时副本',
    currentAsset: '未配置',
  },
  {
    slotKey: 'ancient-heaven',
    name: '远古天庭',
    category: '限时副本',
    currentAsset: '未配置',
  },
]

const UPLOAD_PARAMS = [
  { label: '推荐尺寸', value: '1600 × 684，或同等 21:9 横幅' },
  { label: '文件格式', value: 'WebP 优先，PNG/JPG 可作为源图' },
  { label: '建议大小', value: 'WebP 小于 180KB，源图可保留高清版' },
  { label: '命名规则', value: 'slotKey.webp，例如 kings-war.webp' },
  { label: '显示方式', value: 'center / cover，卡片内墨刷渐隐遮罩' },
  { label: '安全边距', value: '主体放中间，左右边缘留暗，避免压住标题' },
]

// DungeonAssetConfigPanel 渲染副本背景图上传参数占位。
export default function DungeonAssetConfigPanel() {
  return (
    <div className="grid gap-4">
      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)]">
        <div className="flex items-center gap-2 border-b border-[var(--color-border)] px-4 py-3">
          <Image size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">副本图片配置</h2>
          <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">当前为上传参数占位</span>
        </div>
        <div className="grid gap-3 p-4 md:grid-cols-2 xl:grid-cols-3">
          {DUNGEON_IMAGE_CONFIGS.map((item) => (
            <div key={item.slotKey} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
              <div className="mb-2 flex items-center gap-2">
                <span className="text-sm font-bold text-[var(--color-text-primary)]">{item.name}</span>
                <span className="ml-auto rounded-full bg-[var(--color-surface)] px-2 py-0.5 text-[10px] text-[var(--color-text-muted)]">{item.category}</span>
              </div>
              <div className="space-y-1 text-xs text-[var(--color-text-secondary)]">
                <p><span className="text-[var(--color-text-muted)]">slotKey：</span>{item.slotKey}</p>
                <p className="break-all"><span className="text-[var(--color-text-muted)]">当前资源：</span>{item.currentAsset}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)]">
        <div className="flex items-center gap-2 border-b border-[var(--color-border)] px-4 py-3">
          <UploadCloud size={16} className="text-[var(--color-accent)]" />
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">上传参数</h2>
        </div>
        <div className="grid gap-2 p-4 md:grid-cols-2">
          {UPLOAD_PARAMS.map((item) => (
            <div key={item.label} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 py-2">
              <p className="text-[10px] font-semibold text-[var(--color-text-muted)]">{item.label}</p>
              <p className="mt-1 text-xs text-[var(--color-text-secondary)]">{item.value}</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
