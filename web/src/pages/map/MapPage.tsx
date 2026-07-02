/* 本文件实现地图页入口，第一版只展示世界地图玩家城池视图。 */
import type { FC } from 'react'
import WorldMapTab from './components/WorldMapTab'

// MapPage 渲染世界地图主视图，不再把玩家城池地图藏在分类页签中。
const MapPage: FC = () => {
  return <WorldMapTab />
}

export default MapPage
