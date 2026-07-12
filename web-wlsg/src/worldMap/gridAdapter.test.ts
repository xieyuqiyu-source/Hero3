/** 验证真实地图目标到官方错位地块的坐标与图片映射。 */
import { describe, expect, it } from 'vitest'
import { createWorldMapGrid, targetTileImage } from './gridAdapter'
import type { WorldMapTarget, WorldMapViewResponse } from './types'

/** 创建满足接口契约的地图目标。 */
function target(overrides: Partial<WorldMapTarget> = {}): WorldMapTarget {
  return { targetType: 'player_city', targetId: 'p1', playerId: 'p1', name: '许昌', faction: 'wei', relation: 'other', level: 5, status: 'attackable', x: 50, y: 50, distance: 0, direction: 'center', canScout: true, canAttack: true, canPlunder: true, canReinforce: true, ...overrides }
}

/** 创建最小世界地图视野。 */
function view(targets: WorldMapTarget[]): WorldMapViewResponse {
  return { worldId: 'world_1', width: 100, height: 100, self: { playerId: 'p1', worldId: 'world_1', x: 50, y: 50, assignedBy: 'test' }, centerX: 50, centerY: 50, radius: 10, targets, serverTime: '2026-07-12T00:00:00Z' }
}

describe('世界地图网格适配', () => {
  it('生成二十行且按官方七八格交错排列', () => {
    const rows = createWorldMapGrid(view([]))
    expect(rows).toHaveLength(20)
    expect(rows[0]).toHaveLength(7)
    expect(rows[1]).toHaveLength(8)
  })

  it('将真实玩家和黄巾营地放入对应权威坐标', () => {
    const player = target({ relation: 'self' })
    const yellow = target({ targetType: 'yellow_turban', targetId: 'yellow-1', playerId: undefined, name: '黄巾军·魏地', x: 49, y: 49 })
    const cells = createWorldMapGrid(view([player, yellow])).flat()
    expect(cells.find((cell) => cell.key === '50:50')?.target).toEqual(player)
    expect(cells.find((cell) => cell.key === '49:49')?.target).toEqual(yellow)
    expect(targetTileImage(player)).toContain('/3/wei_5.gif')
    expect(targetTileImage(yellow)).toContain('/0/wei_4.gif')
  })

  it('贴近世界边界时所有格子仍保持合法坐标', () => {
    const edge = view([])
    edge.centerX = 0
    edge.centerY = 0
    const cells = createWorldMapGrid(edge).flat()
    expect(Math.min(...cells.map((cell) => cell.x))).toBe(0)
    expect(Math.min(...cells.map((cell) => cell.y))).toBe(0)
    expect(Math.max(...cells.map((cell) => cell.y))).toBe(19)
  })

  it('缩小时增加可见行列且放大时减少地块数量', () => {
    const zoomedOut = createWorldMapGrid(view([]), .75)
    const zoomedIn = createWorldMapGrid(view([]), 1.5)
    expect(zoomedOut.length).toBeGreaterThan(20)
    expect(zoomedOut[0].length).toBeGreaterThan(zoomedIn[0].length)
    expect(zoomedIn.length).toBeLessThan(20)
  })
})
