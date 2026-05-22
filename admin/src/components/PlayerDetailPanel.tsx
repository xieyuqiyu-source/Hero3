import { useCallback, useEffect, useState } from 'react'
import { adminApi } from '@/api/admin'
import type { GameState } from '@/types'
import ResourceAdjustForm from '@/components/ResourceAdjustForm'

interface PlayerDetailPanelProps {
  playerId: string
  onClose: () => void
}

export default function PlayerDetailPanel({ playerId, onClose }: PlayerDetailPanelProps) {
  const [state, setState] = useState<GameState | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadState = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await adminApi.getPlayerState(playerId)
      setState(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [playerId])

  useEffect(() => {
    void loadState()
  }, [loadState])

  const handleAdjustSuccess = (nextState: GameState) => {
    setState(nextState)
  }

  if (loading) {
    return (
      <div className="player-detail-overlay">
        <div className="player-detail-panel">
          <p>加载中...</p>
        </div>
      </div>
    )
  }

  if (error || !state) {
    return (
      <div className="player-detail-overlay">
        <div className="player-detail-panel">
          <p className="inline-error">{error ?? '数据为空'}</p>
          <button type="button" onClick={onClose}>关闭</button>
        </div>
      </div>
    )
  }

  return (
    <div className="player-detail-overlay">
      <div className="player-detail-panel">
        <header className="detail-header">
          <div>
            <h3>{state.player.nickname}</h3>
            <span className="detail-meta">{state.player.id} · {state.player.faction}</span>
          </div>
          <button type="button" className="ghost-button" onClick={onClose}>关闭</button>
        </header>

        <section className="detail-section">
          <h4>资源</h4>
          <div className="detail-grid">
            {Object.entries(state.resources.items).map(([res, amount]) => (
              <div className="detail-stat" key={res}>
                <span>{res}</span>
                <strong>{amount.toLocaleString()}</strong>
                <small>/ {state.resources.capacity[res]?.toLocaleString() ?? '--'}</small>
              </div>
            ))}
          </div>
          <div className="detail-production">
            <span>产量：</span>
            {Object.entries(state.resourceProduction).map(([res, rate]) => (
              <span key={res}>{res} {rate}/h</span>
            ))}
          </div>
        </section>

        <ResourceAdjustForm playerId={playerId} onSuccess={handleAdjustSuccess} />

        <section className="detail-section">
          <h4>建筑 ({state.buildings.length})</h4>
          <div className="detail-building-list">
            {state.buildings.map((b) => (
              <span key={b.id} className="building-chip">
                {b.type} Lv.{b.level}
                {b.upgradeEndsAt && ' ⏳'}
              </span>
            ))}
          </div>
        </section>

        <section className="detail-section">
          <h4>军队</h4>
          <div className="detail-grid">
            {state.army.map((unit) => (
              <div className="detail-stat" key={unit.unitType}>
                <span>{unit.unitType}</span>
                <strong>{unit.amount.toLocaleString()}</strong>
              </div>
            ))}
          </div>
        </section>

        {state.recruitQueues.length > 0 && (
          <section className="detail-section">
            <h4>征兵队列</h4>
            {state.recruitQueues.map((q) => (
              <div key={q.id} className="detail-stat">
                <span>{q.unitType} ×{q.amount}</span>
                <small>{q.status} · {new Date(q.endsAt).toLocaleString()}</small>
              </div>
            ))}
          </section>
        )}

        {state.mapTargets.length > 0 && (
          <section className="detail-section">
            <h4>地图目标</h4>
            {state.mapTargets.map((t) => (
              <div key={t.id} className="detail-stat">
                <span>{t.type} Lv.{t.level}</span>
                <small>战力 {t.power}</small>
              </div>
            ))}
          </section>
        )}

        <section className="detail-section">
          <h4>服务器时间</h4>
          <small>{state.serverTime}</small>
        </section>
      </div>
    </div>
  )
}
