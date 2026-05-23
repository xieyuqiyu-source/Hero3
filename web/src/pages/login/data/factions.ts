import type { Faction } from '../types'

/** 阵营 UI 配置（颜色、文案等前端展示用） */
export interface FactionUIConfig {
  key: Faction
  name: string
  subtitle: string
  color: string
  borderActive: string
  motto: string
  desc: string
  traits: string[]
}

export const FACTION_UI: FactionUIConfig[] = [
  {
    key: 'wei',
    name: '魏',
    subtitle: '曹魏',
    color: 'text-blue-400',
    borderActive: 'border-blue-400',
    motto: '宁教我负天下人，休教天下人负我',
    desc: '挟天子以令诸侯，坐拥中原沃土。曹魏以铁腕治军、唯才是举闻名于世，麾下谋臣如雨、猛将如云，兵精粮足，雄踞北方。',
    traits: ['兵力恢复加成', '谋略系武将强化', '资源产出稳定'],
  },
  {
    key: 'shu',
    name: '蜀',
    subtitle: '蜀汉',
    color: 'text-green-400',
    borderActive: 'border-green-400',
    motto: '勿以恶小而为之，勿以善小而不为',
    desc: '兴复汉室，还于旧都。蜀汉以仁义立国，桃园结义传为佳话，五虎上将威震四方，卧龙凤雏运筹帷幄，虽偏安一隅却志在天下。',
    traits: ['武将忠诚度加成', '步兵系战力强化', '防御建筑加成'],
  },
  {
    key: 'wu',
    name: '吴',
    subtitle: '东吴',
    color: 'text-red-400',
    borderActive: 'border-red-400',
    motto: '内事不决问张昭，外事不决问周瑜',
    desc: '据江东六郡以观天下，凭长江天险与水战之利鼎足三分。东吴英才辈出，水军无敌，火攻赤壁名垂青史，以少胜多屡建奇功。',
    traits: ['水军战力加成', '弓兵系强化', '贸易收入加成'],
  },
]
