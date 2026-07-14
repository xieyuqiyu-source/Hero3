/** 军事战争页静态文案与官方资源映射：不包含任何玩家演示数值。 */
import type { WorldMapMarchAction } from '../game/types'

export interface MilitaryWarOrderContent {
  id: WorldMapMarchAction
  label: string
  image: string
  description: string
}

export const militaryWarContent = {
  tabs: ['调兵遣将', '战争动态', '战争保险'],
  description: '到了实战的时间，派遣军队进攻对手，创造自己的辉煌霸业，千万不要手软。攻击、掠夺、侦察、增援，这四项指令请根据说明合理利用，来击败对手。无论何时，都不要忽略部队的扩充，保证本方在进攻和防守时的有效力量。',
  officialWarning: '注意：当您城池粮食生产力<0，此时粮食囤积量变为0时，城池中驻扎的军队会开始饿死。《饿死规则》详情请参见帮助—军事',
  orders: [
    { id: 'attack', label: '攻击', image: 'db_gj.gif', description: '选择真实玩家城池并派遣部队发动攻击。' },
    { id: 'plunder', label: '掠夺', image: 'db_ld.gif', description: '选择真实玩家城池并派兵掠夺资源。' },
    { id: 'scout', label: '侦察', image: 'db_zc.gif', description: '自动派出本城全部可用侦察兵。' },
    { id: 'reinforce', label: '增援', image: 'db_zy.gif', description: '选择可增援城池并派遣真实兵力与武将。' },
  ] as MilitaryWarOrderContent[],
  unsupportedSections: [
    { id: 'stronghold', title: '本城派遣到据点的军队', message: '现有后端暂未提供据点派遣列表。' },
    { id: 'arena', title: '本城派遣到小队竞技的军队', message: '现有后端暂未提供小队竞技派遣列表。' },
    { id: 'captured', title: '本城被俘虏于他城的军队', message: '现有后端暂未提供被俘军队列表。' },
  ],
}
