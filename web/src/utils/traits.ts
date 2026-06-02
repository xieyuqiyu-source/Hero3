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
    bullets: { label: string; text: string }[]  // 关键效果说明
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
        { label: '触发', text: '主动攻击时，在战斗开始前触发' },
        { label: '效果', text: '俘虏敌方部分兵力，俘虏成功后加入己方军队' },
        { label: '规则', text: '俘虏数量受俘虏比例和单兵种上限影响' },
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
        { label: '触发', text: '被敌方侦查时概率触发' },
        { label: '效果', text: '侦查报告可能显示伪装后的兵力、资源或城防信息' },
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
        { label: '触发', text: '本次出征只使用虎豹骑时概率触发' },
        { label: '效果', text: '触发后突破敌方防守类将领特性' },
        { label: '限制', text: '混编其他兵种时不触发' },
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
        { label: '触发', text: '主动进攻胜利后概率触发' },
        { label: '效果', text: '目标城池进入压制状态，恢复和发展效率下降' },
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
        { label: '触发', text: '己方防守时概率触发' },
        { label: '效果', text: '本次战斗不产生兵力伤亡' },
        { label: '结果', text: '进攻方仍可获得部分资源，但无法造成杀兵收益' },
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
        { label: '触发', text: '战斗结束后概率触发' },
        { label: '效果', text: '己方本场阵亡兵按比例复活，并回到军队' },
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
        { label: '触发', text: '战斗开始前概率触发' },
        { label: '效果', text: '敌方部分兵力不参与本场攻防计算' },
        { label: '说明', text: '被震慑的兵不会死亡，战后仍回到敌方军队' },
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
        { label: '触发', text: '建筑升级或征兵消耗资源时概率触发' },
        { label: '效果', text: '返还本次消耗的一部分资源' },
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
        { label: '触发', text: '主动攻击时，在战斗结算后概率触发' },
        { label: '效果', text: '对敌方各兵种追加一定比例的额外损失' },
        { label: '规则', text: '额外损失不会超过敌方该兵种总数' },
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
        { label: '触发', text: '出征时概率触发' },
        { label: '效果', text: '减少本次行军时间，更快抵达目标' },
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
        { label: '触发', text: '己方防守时概率触发' },
        { label: '效果', text: '大幅降低敌方本次可掠夺资源' },
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
        { label: '触发', text: '按时间自动累计' },
        { label: '效果', text: '自动产出玩家设置的兵种，不消耗基础资源' },
        { label: '说明', text: '不同兵种需要的产出时间不同' },
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
