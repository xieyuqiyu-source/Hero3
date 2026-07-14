/** 将后端世界坐标映射为官方 20 行错位地图地块。 */
import type { WorldMapTarget, WorldMapViewResponse } from './types'

export interface WorldMapCell {
  key: string
  x: number
  y: number
  row: number
  column: number
  target: WorldMapTarget | null
  image: string
}

const terrainImages = ['bg1.gif', 'bgA3.gif', 'bgA7.gif', 'bgA8.gif']

/** 选择官方关系色城池或黄巾营地图片。 */
export function targetTileImage(target: WorldMapTarget): string {
  const faction = ['wei', 'shu', 'wu'].includes(target.faction) ? target.faction : 'wei'
  if (target.targetType === 'yellow_turban') return `/assets/official/map/newmap/0/${faction}_4.gif`
  const relationDirectory = target.relation === 'self' || target.sameAccount ? '3' : target.relation === 'ally' ? '1' : '0'
  return `/assets/official/map/newmap/${relationDirectory}/${faction}_5.gif`
}

/** 创建随中心点和世界边界变化的 20 行地图，不写死目标数量。 */
export function createWorldMapGrid(view: WorldMapViewResponse, zoom = 1): WorldMapCell[][] {
  const targetByCoordinate = new Map(view.targets.map((target) => [`${target.x}:${target.y}`, target]))
  const safeZoom = Math.max(.75, Math.min(1.5, zoom))
  const visibleRows = Math.min(Math.ceil(20 / safeZoom), view.height)
  const startY = Math.max(0, Math.min(view.height - visibleRows, view.centerY - Math.floor(visibleRows / 2)))
  return Array.from({ length: visibleRows }, (_, row) => {
    const baseColumns = Math.ceil(518 / (74 * safeZoom))
    const count = baseColumns + (row % 2 === 0 ? 0 : 1)
    const visibleColumns = Math.min(count, view.width)
    const startX = Math.max(0, Math.min(view.width - visibleColumns, view.centerX - Math.floor(visibleColumns / 2)))
    const y = startY + row
    return Array.from({ length: visibleColumns }, (_, column) => {
      const x = startX + column
      const target = targetByCoordinate.get(`${x}:${y}`) ?? null
      const terrain = terrainImages[Math.abs(x * 17 + y * 31) % terrainImages.length]
      return { key: `${x}:${y}`, x, y, row, column, target, image: target ? targetTileImage(target) : `/assets/official/map/newmap/${terrain}` }
    })
  })
}
