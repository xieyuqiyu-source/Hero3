/* 本文件实现 GM 后台物品配置、背包和流水管理面板。 */
import { useEffect, useMemo, useState } from 'react'
import { CheckCircle2, Database, Package, Plus, RefreshCcw, Save, Trash2 } from 'lucide-react'
import { adminApi } from '@/api/admin'
import type { InventoryView, ItemDefinition, ItemLedgerEntry } from '@/types'

const errorMessage = (error: unknown) => error instanceof Error ? error.message : String(error)

const CATEGORY_OPTIONS = [
  ['resource_pack', '资源包'],
  ['general_exp', '武将经验'],
  ['recruit_ticket', '征募券'],
  ['buff_item', '增益道具'],
  ['pvp_item', 'PVP 道具'],
  ['ticket', '门票'],
  ['material', '材料'],
  ['chest', '宝箱'],
  ['token', '功能凭证'],
  ['currency_pack', '货币包'],
  ['event_item', '活动物品'],
  ['equipment', '装备'],
] as const

const QUALITY_OPTIONS = [
  ['common', '普通'],
  ['rare', '稀有'],
  ['epic', '史诗'],
  ['legendary', '传说'],
  ['mythic', '神话'],
] as const

const CATEGORY_PREFIX: Record<string, string> = {
  resource_pack: 'resource_pack_',
  general_exp: 'general_exp_',
  recruit_ticket: 'recruit_ticket_',
  buff_item: 'buff_',
  pvp_item: 'pvp_',
  ticket: 'ticket_',
  material: 'material_',
  chest: 'chest_',
  token: 'token_',
  currency_pack: 'currency_pack_',
  event_item: 'event_',
  equipment: 'equipment_',
}

const emptyForm = {
  editingId: '',
  category: 'resource_pack',
  quality: 'common',
  name: '',
  suffix: '',
  description: '',
  icon: '',
  usable: true,
  stackable: true,
  maxStack: 999,
  useTarget: 'self',
  confirmOnUse: 'auto',
  effectsText: '[]',
}

type DropPoolReward = {
  type: string
  id?: string
  amount?: number
  weight: number
  dropPoolId?: string
}

type DropPoolSlot = {
  rolls?: number
  items: DropPoolReward[]
}

type DropPoolDefinition = {
  id?: string
  rolls?: number
  items?: DropPoolReward[]
  slots?: DropPoolSlot[]
}

type DropPoolsDraft = Record<string, DropPoolDefinition>

const emptyDropReward = (): DropPoolReward => ({ type: 'item', id: '', amount: 1, weight: 10000 })

export default function ItemsConfigPanel() {
  const [config, setConfig] = useState<Record<string, ItemDefinition>>({})
  const [configText, setConfigText] = useState('{}')
  const [dropPools, setDropPools] = useState<Record<string, unknown>>({})
  const [dropPoolsText, setDropPoolsText] = useState('{}')
  const [selectedDropPoolId, setSelectedDropPoolId] = useState('')
  const [newDropPoolId, setNewDropPoolId] = useState('')
  const [showDropPoolJson, setShowDropPoolJson] = useState(false)
  const [playerId, setPlayerId] = useState('')
  const [itemId, setItemId] = useState('')
  const [refType, setRefType] = useState('')
  const [inventory, setInventory] = useState<InventoryView | null>(null)
  const [ledger, setLedger] = useState<ItemLedgerEntry[]>([])
  const [form, setForm] = useState(emptyForm)
  const [message, setMessage] = useState('')
  const [loading, setLoading] = useState(false)

  const itemOptions = useMemo(() => Object.entries(config).sort((a, b) => a[0].localeCompare(b[0])), [config])
  const dropPoolDraft = dropPools as DropPoolsDraft
  const dropPoolIds = useMemo(() => Object.keys(dropPoolDraft).sort((a, b) => a.localeCompare(b)), [dropPoolDraft])
  const selectedDropPool = selectedDropPoolId ? dropPoolDraft[selectedDropPoolId] : undefined
  const formItemId = form.editingId || `${CATEGORY_PREFIX[form.category] ?? ''}${form.suffix.trim()}`

  const loadConfig = async () => {
    setLoading(true)
    try {
      const [next, nextDropPools] = await Promise.all([
        adminApi.getAdminItemsConfig(),
        adminApi.getDropPoolsConfig(),
      ])
      setConfig(next)
      setConfigText(JSON.stringify(next, null, 2))
      setDropPools(nextDropPools)
      setDropPoolsText(JSON.stringify(nextDropPools, null, 2))
      const nextPoolIds = Object.keys(nextDropPools).sort((a, b) => a.localeCompare(b))
      setSelectedDropPoolId((current) => current && nextDropPools[current] ? current : (nextPoolIds[0] ?? ''))
      setMessage('物品和掉落池配置已刷新')
    } catch (error) {
      setMessage(`失败: ${errorMessage(error)}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadConfig()
  }, [])

  const parseConfig = () => JSON.parse(configText) as Record<string, ItemDefinition>
  const parseDropPools = () => JSON.parse(dropPoolsText) as DropPoolsDraft

  const syncDropPoolsDraft = (next: DropPoolsDraft, nextSelectedId = selectedDropPoolId) => {
    setDropPools(next)
    setDropPoolsText(JSON.stringify(next, null, 2))
    const ids = Object.keys(next).sort((a, b) => a.localeCompare(b))
    setSelectedDropPoolId(nextSelectedId && next[nextSelectedId] ? nextSelectedId : (ids[0] ?? ''))
  }

  const applyDropPoolsJson = () => {
    try {
      const next = parseDropPools()
      syncDropPoolsDraft(next)
      setMessage('已从 JSON 同步到表单')
    } catch (error) {
      setMessage(`JSON 解析失败: ${errorMessage(error)}`)
    }
  }

  const createDropPool = () => {
    const id = newDropPoolId.trim()
    if (!/^[a-z0-9_]+$/.test(id)) {
      setMessage('掉落池 ID 只能包含小写英文、数字和下划线')
      return
    }
    if (dropPoolDraft[id]) {
      setMessage('掉落池 ID 已存在')
      return
    }
    const next = {
      ...dropPoolDraft,
      [id]: { id, slots: [{ items: [emptyDropReward()] }] },
    }
    syncDropPoolsDraft(next, id)
    setNewDropPoolId('')
    setMessage(`已创建掉落池草稿：${id}`)
  }

  const removeDropPool = (poolId: string) => {
    if (!window.confirm(`确认删除掉落池 ${poolId}？已绑定该 ID 的玩法保存时会校验失败。`)) return
    const next = { ...dropPoolDraft }
    delete next[poolId]
    syncDropPoolsDraft(next)
    setMessage(`已删除掉落池草稿：${poolId}`)
  }

  const updateDropPool = (poolId: string, updater: (pool: DropPoolDefinition) => DropPoolDefinition) => {
    const current = dropPoolDraft[poolId]
    if (!current) return
    syncDropPoolsDraft({ ...dropPoolDraft, [poolId]: updater(current) }, poolId)
  }

  const updateDropPoolSlot = (poolId: string, slotIndex: number, updater: (slot: DropPoolSlot) => DropPoolSlot) => {
    updateDropPool(poolId, (pool) => {
      const slots = pool.slots?.length ? [...pool.slots] : [{ rolls: pool.rolls, items: pool.items ?? [] }]
      slots[slotIndex] = updater(slots[slotIndex])
      return { ...pool, rolls: undefined, items: undefined, slots }
    })
  }

  const addDropPoolSlot = (poolId: string) => {
    updateDropPool(poolId, (pool) => ({
      ...pool,
      rolls: undefined,
      items: undefined,
      slots: [...(pool.slots?.length ? pool.slots : [{ rolls: pool.rolls, items: pool.items ?? [] }]), { items: [emptyDropReward()] }],
    }))
  }

  const removeDropPoolSlot = (poolId: string, slotIndex: number) => {
    updateDropPool(poolId, (pool) => {
      const slots = (pool.slots?.length ? pool.slots : [{ rolls: pool.rolls, items: pool.items ?? [] }]).filter((_, index) => index !== slotIndex)
      return { ...pool, rolls: undefined, items: undefined, slots: slots.length ? slots : [{ items: [emptyDropReward()] }] }
    })
  }

  const updateDropPoolReward = (poolId: string, slotIndex: number, rewardIndex: number, updater: (reward: DropPoolReward) => DropPoolReward) => {
    updateDropPoolSlot(poolId, slotIndex, (slot) => {
      const items = [...slot.items]
      items[rewardIndex] = updater(items[rewardIndex])
      return { ...slot, items }
    })
  }

  const addDropPoolReward = (poolId: string, slotIndex: number) => {
    updateDropPoolSlot(poolId, slotIndex, (slot) => ({ ...slot, items: [...slot.items, emptyDropReward()] }))
  }

  const removeDropPoolReward = (poolId: string, slotIndex: number, rewardIndex: number) => {
    updateDropPoolSlot(poolId, slotIndex, (slot) => {
      const items = slot.items.filter((_, index) => index !== rewardIndex)
      return { ...slot, items: items.length ? items : [emptyDropReward()] }
    })
  }

  const validateConfig = async () => {
    setLoading(true)
    try {
      const parsed = parseConfig()
      const result = await adminApi.validateItemsConfig(parsed)
      setMessage(result.ok ? '校验通过' : `校验失败: ${result.error ?? '未知错误'}`)
    } catch (error) {
      setMessage(`校验失败: ${errorMessage(error)}`)
    } finally {
      setLoading(false)
    }
  }

  const saveConfig = async () => {
    if (!window.confirm('确认保存物品配置？保存后会立即影响后续发放和使用校验。')) return
    setLoading(true)
    try {
      const saved = await adminApi.updateItemsConfig(parseConfig())
      setConfig(saved)
      setConfigText(JSON.stringify(saved, null, 2))
      setMessage('物品配置已保存')
    } catch (error) {
      setMessage(`保存失败: ${errorMessage(error)}`)
    } finally {
      setLoading(false)
    }
  }

  const validateDropPools = async () => {
    setLoading(true)
    try {
      const result = await adminApi.validateDropPoolsConfig(parseDropPools())
      setMessage(result.ok ? '掉落池校验通过' : `掉落池校验失败: ${result.error ?? '未知错误'}`)
    } catch (error) {
      setMessage(`掉落池校验失败: ${errorMessage(error)}`)
    } finally {
      setLoading(false)
    }
  }

  const saveDropPools = async () => {
    if (!window.confirm('确认保存掉落池配置？保存后会立即影响后续掉落结算。')) return
    setLoading(true)
    try {
      const saved = await adminApi.updateDropPoolsConfig(parseDropPools())
      setDropPools(saved)
      setDropPoolsText(JSON.stringify(saved, null, 2))
      setMessage('掉落池配置已保存')
    } catch (error) {
      setMessage(`掉落池保存失败: ${errorMessage(error)}`)
    } finally {
      setLoading(false)
    }
  }

  const applyFormToConfig = () => {
    try {
      const itemId = formItemId
      if (!/^[a-z0-9_]+$/.test(itemId)) {
        setMessage('物品 ID 只能包含小写英文、数字和下划线')
        return
      }
      if (!form.name.trim()) {
        setMessage('物品名称不能为空')
        return
      }
      const next = parseConfig()
      const current = form.editingId ? next[form.editingId] : undefined
      if (!form.editingId && next[itemId]) {
        setMessage('物品 ID 已存在')
        return
      }
      next[itemId] = {
        ...(current ?? {}),
        id: itemId,
        name: form.name.trim(),
        description: form.description.trim(),
        category: form.category,
        quality: form.quality,
        icon: form.icon.trim() || undefined,
        usable: form.usable,
        stackable: form.stackable,
        maxStack: form.stackable ? Math.max(1, form.maxStack) : 1,
        bindType: 'bound',
        useTarget: form.useTarget.trim() || 'self',
        confirmOnUse: form.confirmOnUse,
        effects: JSON.parse(form.effectsText),
        metadata: current?.metadata ?? { version: 1 },
      }
      setConfig(next)
      setConfigText(JSON.stringify(next, null, 2))
      setMessage(form.editingId ? `已更新表单草稿：${itemId}` : `已加入表单草稿：${itemId}`)
    } catch (error) {
      setMessage(`表单内容不合法: ${errorMessage(error)}`)
    }
  }

  const editItem = (id: string) => {
    const item = config[id]
    if (!item) return
    const category = item.category ?? item.type ?? 'resource_pack'
    const prefix = CATEGORY_PREFIX[category] ?? ''
    setForm({
      editingId: id,
      category,
      quality: item.quality ?? item.rarity ?? 'common',
      name: item.name,
      suffix: prefix && id.startsWith(prefix) ? id.slice(prefix.length) : id,
      description: item.description,
      icon: item.icon ?? '',
      usable: item.usable,
      stackable: item.stackable,
      maxStack: item.maxStack,
      useTarget: item.useTarget,
      confirmOnUse: item.confirmOnUse ?? 'auto',
      effectsText: JSON.stringify(item.effects ?? [], null, 2),
    })
    setMessage(`正在编辑：${id}`)
  }

  const loadInventory = async () => {
    if (!playerId) return
    setLoading(true)
    try {
      setInventory(await adminApi.getPlayerInventory(playerId))
      setMessage('玩家背包已刷新')
    } catch (error) {
      setMessage(`背包查询失败: ${errorMessage(error)}`)
    } finally {
      setLoading(false)
    }
  }

  const loadLedger = async () => {
    setLoading(true)
    try {
      const page = await adminApi.getItemLedger({ playerId, itemId, refType, limit: 100 })
      setLedger(page.entries)
      setMessage(`物品流水 ${page.total} 条`)
    } catch (error) {
      setMessage(`流水查询失败: ${errorMessage(error)}`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="grid gap-4">
      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_320px]">
        <textarea
          value={configText}
          onChange={(event) => setConfigText(event.target.value)}
          spellCheck={false}
          className="min-h-[420px] rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3 font-mono text-xs text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
        />
        <div className="grid content-start gap-3">
          <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
            <div className="mb-2 flex items-center gap-2 text-sm font-bold text-[var(--color-text-primary)]">
              <Package size={15} className="text-[var(--color-accent)]" />
              物品配置
            </div>
            <div className="mb-3 text-xs text-[var(--color-text-muted)]">当前 {itemOptions.length} 个物品，保存前会执行后端校验。</div>
            <div className="grid grid-cols-3 gap-2">
              <button type="button" onClick={() => void loadConfig()} disabled={loading} className="inline-flex items-center justify-center gap-1 rounded-lg border border-[var(--color-border)] px-2 py-2 text-xs font-bold text-[var(--color-text-secondary)] disabled:opacity-50">
                <RefreshCcw size={13} />刷新
              </button>
              <button type="button" onClick={() => void validateConfig()} disabled={loading} className="inline-flex items-center justify-center gap-1 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-2 py-2 text-xs font-bold text-emerald-600 disabled:opacity-50">
                <CheckCircle2 size={13} />校验
              </button>
              <button type="button" onClick={() => void saveConfig()} disabled={loading} className="inline-flex items-center justify-center gap-1 rounded-lg border border-[var(--color-accent-border)] bg-[var(--color-accent-light)] px-2 py-2 text-xs font-bold text-[var(--color-accent)] disabled:opacity-50">
                <Save size={13} />保存
              </button>
            </div>
          </div>
          <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
            <div className="mb-2 flex items-center gap-2 text-sm font-bold text-[var(--color-text-primary)]">
              <Database size={15} className="text-[var(--color-accent)]" />
              背包和流水
            </div>
            <input value={playerId} onChange={(event) => setPlayerId(event.target.value)} placeholder="玩家 ID" className="mb-2 w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none" />
            <select value={itemId} onChange={(event) => setItemId(event.target.value)} className="mb-2 w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none">
              <option value="">全部物品</option>
              {itemOptions.map(([id, item]) => <option key={id} value={id}>{item.name}</option>)}
            </select>
            <input value={refType} onChange={(event) => setRefType(event.target.value)} placeholder="来源类型，如 item_use" className="mb-2 w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none" />
            <div className="grid grid-cols-2 gap-2">
              <button type="button" onClick={() => void loadInventory()} disabled={loading || !playerId} className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-2 py-2 text-xs font-bold text-blue-600 disabled:opacity-50">查背包</button>
              <button type="button" onClick={() => void loadLedger()} disabled={loading} className="rounded-lg border border-violet-500/30 bg-violet-500/10 px-2 py-2 text-xs font-bold text-violet-600 disabled:opacity-50">查流水</button>
            </div>
          </div>
          {message && <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3 text-xs text-[var(--color-text-secondary)]">{message}</div>}
        </div>
      </div>
      <div className="grid gap-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3 xl:grid-cols-[260px_minmax(0,1fr)_260px]">
        <div className="grid content-start gap-3">
          <div>
            <div className="mb-2 flex items-center gap-2 text-sm font-bold text-[var(--color-text-primary)]">
              <Package size={15} className="text-[var(--color-accent)]" />
              掉落池配置
            </div>
            <div className="text-xs text-[var(--color-text-muted)]">当前 {dropPoolIds.length} 个掉落池</div>
          </div>
          <div className="grid gap-2">
            <input
              value={newDropPoolId}
              onChange={(event) => setNewDropPoolId(event.target.value)}
              placeholder="新掉落池 ID"
              className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none"
            />
            <button type="button" onClick={createDropPool} className="inline-flex items-center justify-center gap-1 rounded-lg border border-[var(--color-accent-border)] bg-[var(--color-accent-light)] px-2 py-2 text-xs font-bold text-[var(--color-accent)]">
              <Plus size={13} />新建掉落池
            </button>
          </div>
          <div className="grid max-h-[360px] gap-1.5 overflow-y-auto pr-1">
            {dropPoolIds.map((poolId) => (
              <button
                key={poolId}
                type="button"
                onClick={() => setSelectedDropPoolId(poolId)}
                className={`rounded-lg border px-3 py-2 text-left text-xs transition ${selectedDropPoolId === poolId ? 'border-[var(--color-accent-border)] bg-[var(--color-accent-light)] text-[var(--color-accent)]' : 'border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-secondary)]'}`}
              >
                <span className="block truncate font-bold">{poolId}</span>
                <span className="mt-0.5 block text-[10px] opacity-70">{(dropPoolDraft[poolId]?.slots?.length ?? 0) || 1} 个槽位</span>
              </button>
            ))}
            {dropPoolIds.length === 0 && <div className="rounded-lg border border-dashed border-[var(--color-border)] px-3 py-8 text-center text-xs text-[var(--color-text-muted)]">暂无掉落池</div>}
          </div>
        </div>

        <div className="min-w-0">
          {selectedDropPool ? (() => {
            const slots = selectedDropPool.slots?.length ? selectedDropPool.slots : [{ rolls: selectedDropPool.rolls, items: selectedDropPool.items ?? [] }]
            return (
              <div className="grid gap-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div>
                    <div className="text-sm font-bold text-[var(--color-text-primary)]">{selectedDropPoolId}</div>
                    <div className="text-[11px] text-[var(--color-text-muted)]">每个槽位独立抽取；权重按同槽位内相对比例计算。</div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button type="button" onClick={() => addDropPoolSlot(selectedDropPoolId)} className="inline-flex items-center gap-1 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-1.5 text-xs font-bold text-emerald-600">
                      <Plus size={12} />槽位
                    </button>
                    <button type="button" onClick={() => removeDropPool(selectedDropPoolId)} className="inline-flex items-center gap-1 rounded-lg border border-red-500/30 bg-red-500/10 px-2.5 py-1.5 text-xs font-bold text-red-600">
                      <Trash2 size={12} />删除池
                    </button>
                  </div>
                </div>
                {slots.map((slot, slotIndex) => {
                  const totalWeight = slot.items.reduce((sum, reward) => sum + (Number(reward.weight) || 0), 0)
                  return (
                    <div key={`${selectedDropPoolId}-slot-${slotIndex}`} className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
                      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                        <div>
                          <div className="text-xs font-bold text-[var(--color-text-primary)]">槽位 {slotIndex + 1}</div>
                          <div className="text-[10px] text-[var(--color-text-muted)]">总权重 {totalWeight.toLocaleString()}</div>
                        </div>
                        <div className="flex items-center gap-2">
                          <label className="grid grid-cols-[auto_64px] items-center gap-1 text-[10px] text-[var(--color-text-muted)]">
                            抽取
                            <input
                              type="number"
                              min={1}
                              value={slot.rolls ?? 1}
                              onChange={(event) => updateDropPoolSlot(selectedDropPoolId, slotIndex, (current) => ({ ...current, rolls: Math.max(1, Number(event.target.value) || 1) }))}
                              className="h-7 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 text-xs text-[var(--color-text-primary)] outline-none"
                            />
                          </label>
                          <button type="button" onClick={() => addDropPoolReward(selectedDropPoolId, slotIndex)} className="inline-flex h-7 items-center gap-1 rounded-lg border border-[var(--color-border)] px-2 text-[10px] font-bold text-[var(--color-text-secondary)]">
                            <Plus size={11} />掉落项
                          </button>
                          <button type="button" onClick={() => removeDropPoolSlot(selectedDropPoolId, slotIndex)} className="grid h-7 w-7 place-items-center rounded-lg text-red-500 hover:bg-red-500/10">
                            <Trash2 size={12} />
                          </button>
                        </div>
                      </div>
                      <div className="grid gap-2">
                        {slot.items.map((reward, rewardIndex) => {
                          const chance = totalWeight > 0 ? ((Number(reward.weight) || 0) / totalWeight) * 100 : 0
                          return (
                            <div key={`${selectedDropPoolId}-slot-${slotIndex}-reward-${rewardIndex}`} className="grid gap-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-2 lg:grid-cols-[96px_minmax(150px,1fr)_80px_90px_72px_28px] lg:items-end">
                              <label className="grid gap-1">
                                <span className="text-[10px] text-[var(--color-text-muted)]">类型</span>
                                <select
                                  value={reward.type}
                                  onChange={(event) => updateDropPoolReward(selectedDropPoolId, slotIndex, rewardIndex, (current) => ({ ...current, type: event.target.value, id: event.target.value === 'none' ? 'none' : current.id, amount: event.target.value === 'none' ? undefined : (current.amount ?? 1) }))}
                                  className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs outline-none"
                                >
                                  <option value="item">物品</option>
                                  <option value="none">空掉落</option>
                                  <option value="drop_pool">子掉落池</option>
                                </select>
                              </label>
                              <label className="grid gap-1">
                                <span className="text-[10px] text-[var(--color-text-muted)]">{reward.type === 'drop_pool' ? '子掉落池' : '物品'}</span>
                                {reward.type === 'none' ? (
                                  <input value="none" disabled className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs text-[var(--color-text-muted)]" />
                                ) : reward.type === 'drop_pool' ? (
                                  <select
                                    value={reward.dropPoolId ?? ''}
                                    onChange={(event) => updateDropPoolReward(selectedDropPoolId, slotIndex, rewardIndex, (current) => ({ ...current, dropPoolId: event.target.value }))}
                                    className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs outline-none"
                                  >
                                    <option value="">选择掉落池</option>
                                    {dropPoolIds.filter((poolId) => poolId !== selectedDropPoolId).map((poolId) => <option key={poolId} value={poolId}>{poolId}</option>)}
                                  </select>
                                ) : (
                                  <select
                                    value={reward.id ?? ''}
                                    onChange={(event) => updateDropPoolReward(selectedDropPoolId, slotIndex, rewardIndex, (current) => ({ ...current, id: event.target.value }))}
                                    className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs outline-none"
                                  >
                                    <option value="">选择物品</option>
                                    {itemOptions.map(([id, item]) => <option key={id} value={id}>{item.name} · {id}</option>)}
                                  </select>
                                )}
                              </label>
                              <label className="grid gap-1">
                                <span className="text-[10px] text-[var(--color-text-muted)]">数量</span>
                                <input
                                  type="number"
                                  min={1}
                                  disabled={reward.type === 'none'}
                                  value={reward.amount ?? 1}
                                  onChange={(event) => updateDropPoolReward(selectedDropPoolId, slotIndex, rewardIndex, (current) => ({ ...current, amount: Math.max(1, Number(event.target.value) || 1) }))}
                                  className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs outline-none disabled:opacity-50"
                                />
                              </label>
                              <label className="grid gap-1">
                                <span className="text-[10px] text-[var(--color-text-muted)]">权重</span>
                                <input
                                  type="number"
                                  min={1}
                                  value={reward.weight}
                                  onChange={(event) => updateDropPoolReward(selectedDropPoolId, slotIndex, rewardIndex, (current) => ({ ...current, weight: Math.max(1, Number(event.target.value) || 1) }))}
                                  className="h-8 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 text-xs outline-none"
                                />
                              </label>
                              <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1.5 text-center text-[11px] font-bold text-[var(--color-accent)]">
                                {chance.toFixed(2)}%
                              </div>
                              <button type="button" onClick={() => removeDropPoolReward(selectedDropPoolId, slotIndex, rewardIndex)} className="grid h-8 w-8 place-items-center rounded-lg text-red-500 hover:bg-red-500/10">
                                <Trash2 size={12} />
                              </button>
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  )
                })}
              </div>
            )
          })() : (
            <div className="flex min-h-[260px] items-center justify-center rounded-xl border border-dashed border-[var(--color-border)] bg-[var(--color-surface)] text-xs text-[var(--color-text-muted)]">选择或新建一个掉落池</div>
          )}
        </div>

        <div className="grid content-start gap-3">
          <div className="grid grid-cols-3 gap-2">
            <button type="button" onClick={() => void loadConfig()} disabled={loading} className="inline-flex items-center justify-center gap-1 rounded-lg border border-[var(--color-border)] px-2 py-2 text-xs font-bold text-[var(--color-text-secondary)] disabled:opacity-50">
              <RefreshCcw size={13} />刷新
            </button>
            <button type="button" onClick={() => void validateDropPools()} disabled={loading} className="inline-flex items-center justify-center gap-1 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-2 py-2 text-xs font-bold text-emerald-600 disabled:opacity-50">
              <CheckCircle2 size={13} />校验
            </button>
            <button type="button" onClick={() => void saveDropPools()} disabled={loading} className="inline-flex items-center justify-center gap-1 rounded-lg border border-[var(--color-accent-border)] bg-[var(--color-accent-light)] px-2 py-2 text-xs font-bold text-[var(--color-accent)] disabled:opacity-50">
              <Save size={13} />保存
            </button>
          </div>
          <div className="rounded-lg border border-amber-500/20 bg-amber-500/10 px-3 py-2 text-[11px] leading-relaxed text-amber-700">
            NPC 层级只绑定掉落池 ID；具体掉什么、概率多少，在这里维护。
          </div>
          <button type="button" onClick={() => setShowDropPoolJson((value) => !value)} className="rounded-lg border border-[var(--color-border)] px-3 py-2 text-xs font-bold text-[var(--color-text-secondary)]">
            {showDropPoolJson ? '收起高级 JSON' : '高级 JSON'}
          </button>
          {showDropPoolJson && (
            <div className="grid gap-2">
              <textarea
                value={dropPoolsText}
                onChange={(event) => setDropPoolsText(event.target.value)}
                spellCheck={false}
                className="min-h-[260px] rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-3 font-mono text-xs text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent-border)]"
              />
              <button type="button" onClick={applyDropPoolsJson} className="rounded-lg border border-[var(--color-border)] px-3 py-2 text-xs font-bold text-[var(--color-text-secondary)]">
                套用 JSON 到表单
              </button>
            </div>
          )}
        </div>
      </div>
      <div className="grid gap-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3 lg:grid-cols-[320px_minmax(0,1fr)]">
        <div>
          <div className="mb-2 flex items-center gap-2 text-sm font-bold text-[var(--color-text-primary)]">
            <Plus size={15} className="text-[var(--color-accent)]" />
            新建 / 编辑物品
          </div>
          <div className="grid gap-2">
            <select value={form.category} disabled={!!form.editingId} onChange={(event) => setForm({ ...form, category: event.target.value })} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none disabled:opacity-60">
              {CATEGORY_OPTIONS.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
            </select>
            <select value={form.quality} onChange={(event) => setForm({ ...form, quality: event.target.value })} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none">
              {QUALITY_OPTIONS.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
            </select>
            <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="物品名称" className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none" />
            <div className="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-2">
              <span className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-2 text-xs text-[var(--color-text-muted)]">{CATEGORY_PREFIX[form.category] ?? ''}</span>
              <input value={form.suffix} disabled={!!form.editingId} onChange={(event) => setForm({ ...form, suffix: event.target.value })} placeholder="ID 后缀" className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none disabled:opacity-60" />
            </div>
            <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 font-mono text-[11px] text-[var(--color-text-secondary)]">{formItemId || '等待填写 ID'}</div>
            <input value={form.icon} onChange={(event) => setForm({ ...form, icon: event.target.value })} placeholder="图标 key" className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none" />
            <input value={form.useTarget} onChange={(event) => setForm({ ...form, useTarget: event.target.value })} placeholder="使用目标" className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none" />
            <textarea value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} placeholder="描述" className="min-h-20 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none" />
            <div className="grid grid-cols-3 gap-2">
              <label className="flex items-center gap-1 text-xs"><input type="checkbox" checked={form.usable} onChange={(event) => setForm({ ...form, usable: event.target.checked })} />可使用</label>
              <label className="flex items-center gap-1 text-xs"><input type="checkbox" checked={form.stackable} onChange={(event) => setForm({ ...form, stackable: event.target.checked, maxStack: event.target.checked ? form.maxStack : 1 })} />可堆叠</label>
              <input type="number" min={1} value={form.maxStack} onChange={(event) => setForm({ ...form, maxStack: Number(event.target.value) || 1 })} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-1 text-xs outline-none" />
            </div>
            <select value={form.confirmOnUse} onChange={(event) => setForm({ ...form, confirmOnUse: event.target.value })} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs outline-none">
              <option value="auto">自动确认</option>
              <option value="always">始终确认</option>
              <option value="never">不确认</option>
            </select>
            <button type="button" onClick={applyFormToConfig} className="rounded-lg border border-[var(--color-accent-border)] bg-[var(--color-accent-light)] px-3 py-2 text-xs font-bold text-[var(--color-accent)]">写入配置草稿</button>
            <button type="button" onClick={() => setForm(emptyForm)} className="rounded-lg border border-[var(--color-border)] px-3 py-2 text-xs font-bold text-[var(--color-text-secondary)]">清空表单</button>
          </div>
        </div>
        <div className="grid gap-3">
          <textarea value={form.effectsText} onChange={(event) => setForm({ ...form, effectsText: event.target.value })} spellCheck={false} className="min-h-[220px] rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-3 font-mono text-xs outline-none" />
          <div className="grid max-h-[260px] gap-2 overflow-y-auto pr-1">
            {itemOptions.map(([id, item]) => (
              <button key={id} type="button" onClick={() => editItem(id)} className="grid grid-cols-[1fr_auto] gap-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-left text-xs">
                <span>
                  <strong className="text-[var(--color-text-primary)]">{item.name}</strong>
                  <span className="ml-2 text-[var(--color-text-muted)]">{id}</span>
                </span>
                <span className="text-[var(--color-text-muted)]">{item.category ?? item.type} / {item.quality ?? item.rarity}</span>
              </button>
            ))}
          </div>
        </div>
      </div>
      {inventory && (
        <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
          <div className="mb-2 text-sm font-bold text-[var(--color-text-primary)]">玩家背包格子 {inventory.inventorySlots?.length ?? 0} / 9999</div>
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
            {(inventory.inventorySlots ?? []).map((slot) => (
              <div key={slot.slotId ?? slot.itemId} className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs">
                <div className="font-bold text-[var(--color-text-primary)]">{config[slot.itemId]?.name ?? slot.itemId}</div>
                <div className="mt-1 text-[var(--color-text-muted)]">{slot.slotId} · x{slot.amount}</div>
              </div>
            ))}
          </div>
        </div>
      )}
      <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface-dim)] p-3">
        <div className="mb-2 text-sm font-bold text-[var(--color-text-primary)]">物品流水</div>
        <div className="grid gap-2">
          {ledger.map((entry) => (
            <div key={entry.id} className="grid gap-1 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-xs md:grid-cols-[1fr_auto_auto] md:items-center">
              <div>
                <div className="font-bold text-[var(--color-text-primary)]">{config[entry.itemId]?.name ?? entry.itemId}</div>
                <div className="text-[var(--color-text-muted)]">{entry.playerId} · {entry.reason} · {entry.refType || '-'}</div>
              </div>
              <div className={entry.changeAmount >= 0 ? 'font-bold text-emerald-600' : 'font-bold text-red-600'}>{entry.changeAmount >= 0 ? '+' : ''}{entry.changeAmount}</div>
              <div className="text-[var(--color-text-muted)]">{entry.beforeAmount} → {entry.afterAmount}</div>
            </div>
          ))}
          {ledger.length === 0 && <div className="text-xs text-[var(--color-text-muted)]">暂无流水</div>}
        </div>
      </div>
    </div>
  )
}
