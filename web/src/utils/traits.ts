/**
 * 将领特性元信息（前端展示用）
 *
 * 后端 GET /api/v1/admin/general-traits 也能返回这个，但客户端硬编码一份能减少请求。
 * 新增特性时同步更新此处。
 */

export interface TraitMeta {
  id: string
  name: string
  /** 简短描述（一行） */
  description: string
  /** 触发时机的中文说明 */
  trigger: string
  /** 玩家可见的图标（emoji） */
  icon: string
  /** 详细说明（弹窗里显示的多段文字） */
  details: {
    summary: string                       // 一句话总结效果
    bullets: { label: string; text: string }[]  // 关键点（爽点 / 克制 / 平衡点 等）
  }
}

export const TRAIT_REGISTRY: Record<string, TraitMeta> = {
  // === 魏国 ===
  meiren: {
    id: 'meiren',
    name: '俘虏',
    description: '进攻后可俘虏敌方部分兵力',
    trigger: '进攻后',
    icon: '🌸',
    details: {
      summary: '进攻后触发，可俘虏敌方部分兵力归己方所用。',
      bullets: [
        { label: '爽点', text: '以战养战，越打越滚雪球' },
        { label: '平衡点', text: '需要限制俘虏比例和单场上限' },
      ],
    },
  },
  yibing: {
    id: 'yibing',
    name: '疑兵',
    description: '被侦查时生成伪装战报',
    trigger: '被侦查时',
    icon: '🎭',
    details: {
      summary: '被侦查时触发，生成伪装战报，显示假的兵力、资源、城防信息。',
      bullets: [
        { label: '爽点', text: '误导敌方判断，让敌人基于假情报做错误决策' },
        { label: '克制', text: '依赖侦查和偷袭的玩法' },
      ],
    },
  },
  huchi: {
    id: 'huchi',
    name: '虎痴',
    description: '纯虎豹骑出征时概率突破敌方防守特性',
    trigger: '纯虎豹骑出征时',
    icon: '🐯',
    details: {
      summary: '纯虎豹骑出征时概率触发，突破敌方一切防守特性。',
      bullets: [
        { label: '爽点', text: '用魏国精锐骑兵硬破空城计、江东固守、疑兵等防守套路' },
        { label: '平衡点', text: '必须纯虎豹骑，兵贵，且概率触发' },
      ],
    },
  },
  weiwuhaoling: {
    id: 'weiwuhaoling',
    name: '魏武号令',
    description: '进攻胜利后概率压制目标城池',
    trigger: '进攻胜利后',
    icon: '👑',
    details: {
      summary: '进攻胜利后概率触发，压制目标城池一段时间（征兵速度下降、建筑速度下降、资源产量下降）。',
      bullets: [
        { label: '爽点', text: '打赢不只是抢资源，还让对方恢复和发展变慢' },
        { label: '克制', text: '恢复型、发育型、防守拖节奏玩法' },
      ],
    },
  },

  // === 蜀国 ===
  kongchengji: {
    id: 'kongchengji',
    name: '空城计',
    description: '防守时概率使战斗无伤亡',
    trigger: '防守时',
    icon: '🏯',
    details: {
      summary: '防守时概率触发，双方兵不死，战斗无实际伤亡。进攻方可以抢到部分资源，但无法打出杀兵收益。',
      bullets: [
        { label: '爽点', text: '敌人来得再快，也可能无功而返' },
        { label: '克制', text: '突袭杀兵、甄宓俘虏、吕蒙闪电战' },
      ],
    },
  },
  rende: {
    id: 'rende',
    name: '仁德',
    description: '己方死亡兵概率复活',
    trigger: '战斗后',
    icon: '🕊️',
    details: {
      summary: '己方死亡兵有概率复活一部分。',
      bullets: [
        { label: '爽点', text: '打消耗战很舒服，损失比别人低' },
        { label: '匹配定位', text: '蜀国"资源消耗少、恢复强"' },
        { label: '平衡点', text: '复活比例不能太高，避免和空城计叠加过强' },
      ],
    },
  },
  zhenshe: {
    id: 'zhenshe',
    name: '震慑',
    description: '开战前概率减少敌方参战兵力',
    trigger: '开战前',
    icon: '😱',
    details: {
      summary: '开战前概率触发，让敌方实际参战兵力减少一定百分比。被震慑的兵不死亡，只是不参与本场攻防计算。',
      bullets: [
        { label: '爽点', text: '敌人大军压境，但开战时被吓退一部分战斗力' },
        { label: '克制', text: '爆兵流、人海战术' },
      ],
    },
  },
  qiaojiang: {
    id: 'qiaojiang',
    name: '巧匠',
    description: '建造或征兵时概率返还资源',
    trigger: '建造 / 征兵时',
    icon: '🛠️',
    details: {
      summary: '建筑升级或征兵时概率触发，返还一部分消耗资源。',
      bullets: [
        { label: '爽点', text: '花资源时有机会返钱，长期发展更省' },
        { label: '匹配定位', text: '蜀国"消耗资源少"' },
        { label: '平衡点', text: '返还概率和比例要控制' },
      ],
    },
  },

  // === 吴国 ===
  huogong: {
    id: 'huogong',
    name: '火攻',
    description: '战斗中对敌方造成百分比伤害',
    trigger: '战斗中',
    icon: '🔥',
    details: {
      summary: '战斗中触发，对敌方造成百分比伤害。',
      bullets: [
        { label: '爽点', text: '低战力玩家也可能打出高额伤害，有越级挑战感' },
        { label: '克制', text: '重兵堆防、屯兵流' },
        { label: '平衡点', text: '需要单场伤害上限，避免小号无成本换大号兵' },
      ],
    },
  },
  baiyidujiang: {
    id: 'baiyidujiang',
    name: '白衣渡江',
    description: '出征时概率减少行军时间',
    trigger: '出征时',
    icon: '⚡',
    details: {
      summary: '出征时概率触发"闪电战"，减少本次行军时间。',
      bullets: [
        { label: '爽点', text: '缩短敌方预警时间，适合突袭、掠夺、打时间差' },
        { label: '克制', text: '普通种田玩家、反应慢的防守方' },
        { label: '被克制', text: '诸葛亮空城计、孙权江东固守、司马懿疑兵' },
      ],
    },
  },
  jiangdong: {
    id: 'jiangdong',
    name: '江东固守',
    description: '防守时大概率降低敌方资源掠夺',
    trigger: '防守时',
    icon: '🛡️',
    details: {
      summary: '防守时大概率触发，敌方资源掠夺大幅降低，甚至接近抢不到。',
      bullets: [
        { label: '爽点', text: '别人可以打你，但很难从你这里拿走东西' },
        { label: '克制', text: '吕蒙闪电战、突袭抢资源流' },
        { label: '匹配定位', text: '吴国"防御强"' },
      ],
    },
  },
  guangjiliang: {
    id: 'guangjiliang',
    name: '广积粮',
    description: '每隔一段时间自动产出指定兵种',
    trigger: '长期屯兵',
    icon: '🌾',
    details: {
      summary: '每隔一段时间自动产出指定兵种，不消耗资源只消耗时间。兵种可由玩家设置，不同兵种产出时间不同。',
      bullets: [
        { label: '爽点', text: '不频繁上线也能慢慢积累兵力' },
        { label: '平衡点', text: '需要产兵上限、切换冷却或领取限制' },
      ],
    },
  },
}

export function getTraitMeta(id: string): TraitMeta {
  return TRAIT_REGISTRY[id] ?? {
    id,
    name: id,
    description: '',
    trigger: '',
    icon: '⚔️',
    details: { summary: '', bullets: [] },
  }
}

/** 将领 ID → 该将领拥有的特性 ID 列表 */
export const GENERAL_TRAITS: Record<string, string[]> = {
  // 魏国
  zhenmi: ['meiren'],
  simayi: ['yibing'],
  xuchu: ['huchi'],
  caocao: ['weiwuhaoling'],
  // 蜀国
  zhugeliang: ['kongchengji'],
  liubei: ['rende'],
  zhangfei: ['zhenshe'],
  huangyueying: ['qiaojiang'],
  // 吴国
  zhouyu: ['huogong'],
  lvmeng: ['baiyidujiang'],
  sunquan: ['jiangdong'],
  lusu: ['guangjiliang'],
}

export function getGeneralTraits(generalId: string): TraitMeta[] {
  const traitIds = GENERAL_TRAITS[generalId] ?? []
  return traitIds.map(getTraitMeta)
}

/** 把 params 中常见的 key 翻译成中文 */
const PARAM_LABELS: Record<string, string> = {
  captureRate: '俘虏比例',
  captureMax: '单兵种上限',
  triggerChance: '触发概率',
  damagePercent: '额外伤害',
  reviveRate: '复活比例',
}

export function formatParamLabel(key: string): string {
  return PARAM_LABELS[key] ?? key
}

/** 把 params 值格式化（百分比 / 绝对值） */
export function formatParamValue(key: string, value: number): string {
  if (key.endsWith('Rate') || key.endsWith('Chance') || key.endsWith('Percent')) {
    return `${Math.round(value * 100)}%`
  }
  return value.toLocaleString()
}
