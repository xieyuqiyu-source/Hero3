/* 本文件展示 GM 后续配置副本背景图所需的上传参数。 */
import { useEffect, useState } from 'react'
import { Flame, Image, RefreshCcw, Save, UploadCloud } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { ReincarnationConfig, ReincarnationRun } from '@/types'

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
      <ReincarnationAdminPanel />
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

// ReincarnationAdminPanel 管理轮回绝境配置和异常实例。
function ReincarnationAdminPanel() {
  const [configText, setConfigText] = useState('')
  const [runs, setRuns] = useState<ReincarnationRun[]>([])
  const [playerId, setPlayerId] = useState('')
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    void loadConfig()
    void loadRuns()
  }, [])

  // loadConfig 读取轮回绝境 JSON 配置。
  const loadConfig = async () => {
    setLoading(true)
    try {
      const config = await adminApi.getReincarnationConfig()
      setConfigText(JSON.stringify(config, null, 2))
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '轮回配置加载失败')
    } finally {
      setLoading(false)
    }
  }

  // loadRuns 读取轮回绝境实例列表。
  const loadRuns = async () => {
    try {
      const result = await adminApi.listReincarnationRuns(playerId, 50, 0)
      setRuns(result.items)
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '轮回实例加载失败')
    }
  }

  // saveConfig 保存轮回绝境 JSON 配置。
  const saveConfig = async () => {
    setLoading(true)
    try {
      const parsed = JSON.parse(configText) as ReincarnationConfig
      const saved = await adminApi.updateReincarnationConfig(parsed)
      setConfigText(JSON.stringify(saved, null, 2))
      setMessage('轮回绝境配置已保存')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '轮回配置保存失败')
    } finally {
      setLoading(false)
    }
  }

  // runAction 执行实例修复操作后刷新列表。
  const runAction = async (runId: string, action: 'settle' | 'repair') => {
    setLoading(true)
    try {
      if (action === 'settle') await adminApi.forceSettleReincarnationRun(runId)
      else await adminApi.repairReincarnationReward(runId)
      setMessage(action === 'settle' ? '已强制结算实例' : '已修复奖励状态')
      await loadRuns()
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '操作失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="flex items-center gap-2 border-b border-[var(--color-border)] px-4 py-3">
        <Flame size={16} className="text-[var(--color-accent)]" />
        <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">轮回绝境</h2>
        <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">配置与实例处理</span>
      </div>
      <div className="grid gap-4 p-4 xl:grid-cols-[1.1fr_1fr]">
        <div className="grid gap-3">
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => void loadConfig()}
              disabled={loading}
              className="inline-flex h-9 items-center gap-1.5 rounded-xl border border-[var(--color-border)] px-3 text-xs font-bold text-[var(--color-text-secondary)] disabled:opacity-50"
            >
              <RefreshCcw size={13} />
              刷新配置
            </button>
            <button
              type="button"
              onClick={() => void saveConfig()}
              disabled={loading}
              className="inline-flex h-9 items-center gap-1.5 rounded-xl bg-[var(--color-accent)] px-3 text-xs font-bold text-white disabled:opacity-50"
            >
              <Save size={13} />
              保存配置
            </button>
          </div>
          <textarea
            value={configText}
            onChange={(event) => setConfigText(event.target.value)}
            className="min-h-[360px] rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3 font-mono text-xs text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
            spellCheck={false}
          />
        </div>

        <div className="grid content-start gap-3">
          <div className="flex gap-2">
            <input
              value={playerId}
              onChange={(event) => setPlayerId(event.target.value)}
              placeholder="按 playerId 筛选"
              className="h-9 min-w-0 flex-1 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-3 text-xs text-[var(--color-text-primary)] outline-none"
            />
            <button
              type="button"
              onClick={() => void loadRuns()}
              className="h-9 rounded-xl border border-[var(--color-border)] px-3 text-xs font-bold text-[var(--color-text-secondary)]"
            >
              查询
            </button>
          </div>
          {message && <p className="rounded-xl border border-amber-500/25 bg-amber-500/8 px-3 py-2 text-xs text-amber-700">{message}</p>}
          <div className="grid gap-2">
            {runs.length === 0 ? (
              <p className="rounded-xl border border-dashed border-[var(--color-border)] px-3 py-8 text-center text-xs text-[var(--color-text-muted)]">暂无轮回实例</p>
            ) : runs.map((run) => (
              <div key={run.id} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-bold text-[var(--color-text-primary)]">{run.levelName}</span>
                  <span className="rounded-full bg-[var(--color-surface)] px-2 py-0.5 text-[10px] text-[var(--color-text-muted)]">{run.status}</span>
                  <span className="ml-auto text-[10px] text-[var(--color-text-muted)]">第 {run.currentWave}/18 波</span>
                </div>
                <p className="mt-1 break-all text-[10px] text-[var(--color-text-muted)]">{run.id} · {run.playerId}</p>
                <div className="mt-3 flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => void runAction(run.id, 'settle')}
                    disabled={loading}
                    className="h-8 rounded-lg border border-amber-500/30 bg-amber-500/10 px-2 text-[11px] font-bold text-amber-700 disabled:opacity-50"
                  >
                    强制结算
                  </button>
                  <button
                    type="button"
                    onClick={() => void runAction(run.id, 'repair')}
                    disabled={loading || run.status === 'running'}
                    className="h-8 rounded-lg border border-green-500/30 bg-green-500/10 px-2 text-[11px] font-bold text-green-700 disabled:opacity-50"
                  >
                    修复奖励
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
