/**
 * 将领双特性元信息。
 *
 * 后端 GM 接口也会返回注册表；玩家侧保留一份静态文案用于登录页和将领详情快速展示。
 */

export interface TraitMeta {
  id: string
  name: string
  traitType: 'special' | 'bonus'
  description: string
  trigger: string
  icon: string
  details: {
    summary: string
    bullets: { label: string; text: string }[]
  }
}

export type TraitOutcomeValue = number | string | Record<string, number>

export interface TraitOutcomeFormatOptions {
  faction?: string
  units?: Record<string, Record<string, { name?: string }>>
  sortUnitEntries?: (troops?: Record<string, number>, faction?: string, units?: Record<string, Record<string, { name?: string }>>) => Array<[string, number]>
}

const trait = (id: string, name: string, traitType: 'special' | 'bonus', description: string, trigger: string): TraitMeta => ({
  id,
  name,
  traitType,
  description,
  trigger,
  icon: traitType === 'special' ? '✦' : '◆',
  details: {
    summary: description,
    bullets: [
      { label: '类型', text: traitType === 'special' ? '特殊特性' : '加成特性' },
      { label: '触发', text: trigger },
      { label: '配置', text: '具体数值由 GM 参数控制，战报只显示实际触发结果' },
    ],
  },
})

export const TRAIT_REGISTRY: Record<string, TraitMeta> = {
  weiwu_haoling: trait('weiwu_haoling', '魏武号令', 'special', '按时间强化虎卫或特殊兵征兵进度。', '资源/征兵结算时'),
  weiwu_tongyu: trait('weiwu_tongyu', '魏武统御', 'bonus', '虎卫或特殊兵攻防提升。', '战斗前'),
  yibing_touxi: trait('yibing_touxi', '疑兵偷袭', 'special', '战前偷袭削弱敌方兵力。', '战斗前'),
  mouding_houfa: trait('mouding_houfa', '谋定后发', 'bonus', '削弱敌方攻击爆发。', '战斗前'),
  meiren: trait('meiren', '俘虏敌兵', 'special', '战斗开始前俘虏敌方部分兵力。', '战斗前'),
  meihuo_raozhen: trait('meihuo_raozhen', '魅惑扰阵', 'bonus', '降低敌方攻防表现。', '战斗前'),
  huchi_chongzhen: trait('huchi_chongzhen', '虎痴冲阵', 'special', '概率突破敌方防守类加成。', '战斗前'),
  pojun_pofang: trait('pojun_pofang', '破敌防御', 'bonus', '敌方防御降低。', '战斗前'),
  huzhu_sizhan: trait('huzhu_sizhan', '护主死战', 'special', '濒临失败时降低最终损失。', '战斗结束后'),
  sizhandaodi: trait('sizhandaodi', '死战到底', 'bonus', '步兵攻击提升。', '战斗前'),
  jixing_benxi: trait('jixing_benxi', '疾行奔袭', 'special', '出征或增援行军时间缩短。', '行军创建时'),
  dunzhen_fangyu: trait('dunzhen_fangyu', '盾阵防御', 'bonus', '我军防御提升。', '战斗前'),
  weizhen_zhenhe: trait('weizhen_zhenhe', '威震震慑', 'special', '以少打多时让敌方部分兵不参战。', '战斗前'),
  weizhen_xiaoyao: trait('weizhen_xiaoyao', '威震逍遥', 'bonus', '骑兵攻击提升。', '战斗前'),
  shengui_zhicai: trait('shengui_zhicai', '神鬼之才', 'special', '征兵资源消耗降低。', '征兵消耗时'),
  guicai_yice: trait('guicai_yice', '鬼才遗策', 'bonus', '战败时己方最终损失降低。', '战斗结束后'),
  wangzuo_zhicai: trait('wangzuo_zhicai', '王佐之才', 'special', '资源、建筑、征兵效率提升。', '资源/征兵结算时'),
  neizheng_jingying: trait('neizheng_jingying', '内政精营', 'bonus', '资源产量提升。', '资源结算时'),
  rende: trait('rende', '仁德天下', 'special', '战后复活本场阵亡士兵。', '战斗结束后'),
  renzhu_shouhu: trait('renzhu_shouhu', '仁主守护', 'bonus', '己方最终损失降低。', '战斗结束后'),
  shuiyan_qijun: trait('shuiyan_qijun', '水淹七军', 'special', '战前水淹削弱敌军。', '战斗前'),
  wusheng_pojun: trait('wusheng_pojun', '武圣破军', 'bonus', '主力攻击提升。', '战斗前'),
  zhenhe_quanjun: trait('zhenhe_quanjun', '震慑全军', 'special', '概率让敌军部分兵不参战。', '战斗前'),
  wanren_nuhou: trait('wanren_nuhou', '万人怒吼', 'bonus', '步兵攻击提升。', '战斗前'),
  qimen_dunjia: trait('qimen_dunjia', '奇门遁甲', 'special', '困住敌军，降低参战兵力。', '战斗前'),
  wolong_mouzhi: trait('wolong_mouzhi', '卧龙谋制', 'bonus', '降低敌方特性触发率。', '战斗中'),
  longdan_jiuyuan: trait('longdan_jiuyuan', '龙胆救援', 'special', '增援或防守时保护己方损失。', '战斗结算后'),
  qijin_qichu: trait('qijin_qichu', '七进七出', 'bonus', '全军速度提升。', '行军创建时'),
  xiliang_tuji: trait('xiliang_tuji', '西凉突击', 'special', '骑兵开战追加冲锋伤害。', '战斗结算后'),
  tianshen_xiafan: trait('tianshen_xiafan', '天神下凡', 'bonus', '武将攻击固定提升。', '战斗前'),
  baibu_chuanyang: trait('baibu_chuanyang', '百步穿杨', 'special', '概率使敌方失去防御加成。', '战斗前'),
  laodang_yizhuang: trait('laodang_yizhuang', '老当益壮', 'bonus', '对高口粮兵伤害提升。', '战斗结算后'),
  qibing_raohou: trait('qibing_raohou', '奇兵绕后', 'special', '主动进攻时绕过部分防御。', '战斗前'),
  gushou_hanzhong: trait('gushou_hanzhong', '固守汉中', 'bonus', '武将防御固定提升。', '战斗前'),
  jiangdong_haoling: trait('jiangdong_haoling', '江东号令', 'special', '防守失败时降低敌方掠夺收益。', '掠夺结算时'),
  jiangdong_gushou: trait('jiangdong_gushou', '江东固守', 'bonus', '全军防御提升。', '战斗前'),
  xiaobawang_zhuiji: trait('xiaobawang_zhuiji', '小霸王追击', 'special', '胜利后追加追击损失。', '战斗结算后'),
  xiaobawang_tieqi: trait('xiaobawang_tieqi', '小霸王', 'bonus', '霸王骑攻击固定提升。', '战斗前'),
  huogong: trait('huogong', '火烧赤壁', 'special', '战斗中烧死敌方部分兵力。', '战斗结算后'),
  meizhoulang_junlue: trait('meizhoulang_junlue', '美周郎军略', 'bonus', '火攻伤害或全军攻击提升。', '战斗前'),
  huoshao_lianying: trait('huoshao_lianying', '火烧联营', 'special', '概率烧死敌方步兵。', '战斗结算后'),
  lianying_zengshang: trait('lianying_zengshang', '连营增伤', 'bonus', '对步兵伤害提升。', '战斗结算后'),
  baiyi_dujiang: trait('baiyi_dujiang', '白衣渡江', 'special', '概率隐秘行踪并加快行军。', '行军创建时'),
  baiyi_jixing: trait('baiyi_jixing', '白衣急行', 'bonus', '队伍速度提升。', '行军创建时'),
  kuairu_shandian: trait('kuairu_shandian', '快如闪电', 'special', '概率触发闪电战，极大缩短行军时间。', '行军创建时'),
  xinyi_yonglie: trait('xinyi_yonglie', '信义勇烈', 'bonus', '将领或援军攻击提升。', '战斗前'),
  jinfan_jielue: trait('jinfan_jielue', '锦帆劫掠', 'special', '掠夺收益提升。', '掠夺结算时'),
  jinfan_qixi: trait('jinfan_qixi', '锦帆奇袭', 'bonus', '掠夺战攻击提升。', '战斗前'),
  kurouji: trait('kurouji', '苦肉计', 'special', '概率压制敌方特性。', '战斗中'),
  kurou_fanji: trait('kurou_fanji', '苦肉反击', 'bonus', '承受代价后提高敌方损失。', '战斗结算后'),
}

export const GENERAL_TRAITS: Record<string, string[]> = {
  caocao: ['weiwu_haoling', 'weiwu_tongyu'],
  simayi: ['yibing_touxi', 'mouding_houfa'],
  zhenmi: ['meiren', 'meihuo_raozhen'],
  xuchu: ['huchi_chongzhen', 'pojun_pofang'],
  dianwei: ['huzhu_sizhan', 'sizhandaodi'],
  xiahouyuan: ['jixing_benxi', 'dunzhen_fangyu'],
  zhangliao: ['weizhen_zhenhe', 'weizhen_xiaoyao'],
  guojia: ['shengui_zhicai', 'guicai_yice'],
  xunyu: ['wangzuo_zhicai', 'neizheng_jingying'],
  liubei: ['rende', 'renzhu_shouhu'],
  guanyu: ['shuiyan_qijun', 'wusheng_pojun'],
  zhangfei: ['zhenhe_quanjun', 'wanren_nuhou'],
  zhugeliang: ['qimen_dunjia', 'wolong_mouzhi'],
  zhaoyun: ['longdan_jiuyuan', 'qijin_qichu'],
  machao: ['xiliang_tuji', 'tianshen_xiafan'],
  huangzhong: ['baibu_chuanyang', 'laodang_yizhuang'],
  weiyan: ['qibing_raohou', 'gushou_hanzhong'],
  sunquan: ['jiangdong_haoling', 'jiangdong_gushou'],
  sunce: ['xiaobawang_zhuiji', 'xiaobawang_tieqi'],
  zhouyu: ['huogong', 'meizhoulang_junlue'],
  luxun: ['huoshao_lianying', 'lianying_zengshang'],
  lvmeng: ['baiyi_dujiang', 'baiyi_jixing'],
  taishici: ['kuairu_shandian', 'xinyi_yonglie'],
  ganning: ['jinfan_jielue', 'jinfan_qixi'],
  huanggai: ['kurouji', 'kurou_fanji'],
}

export function getTraitMeta(id: string): TraitMeta {
  return TRAIT_REGISTRY[id] ?? trait(id, id, 'special', '', '')
}

export function getGeneralTraits(generalId: string): TraitMeta[] {
  return (GENERAL_TRAITS[generalId] ?? []).map(getTraitMeta)
}

const PARAM_LABELS: Record<string, string> = {
  triggerChance: '触发概率',
  effectRate: '效果比例',
  captureRate: '俘虏比例',
  captureMax: '单兵种上限',
  maxCapturePerUnit: '单兵种俘虏上限',
  resourceCostReduction: '资源消耗降低',
  guardPerMinute: '每分钟产兵',
  maxGuardPerDay: '单次产兵上限',
  attackBonusRate: '攻击加成',
  attackReductionRate: '攻击降低',
  defenseBonusRate: '防御加成',
  speedBonusRate: '速度加成',
  generalAttackFlat: '武将攻击',
  generalDefenseFlat: '武将防御',
  unitAttackFlat: '兵种攻击',
  enemyDefenseReductionRate: '敌方防御降低',
  lossReductionRate: '损失降低',
  reviveRate: '复活比例',
  maxReviveCount: '复活上限',
  returnRate: '返还比例',
  maxReturnCount: '返还上限',
  plunderBonusRate: '掠夺修正',
  disableTraitRate: '压制比例',
  disableTraitCount: '压制特性数',
  damagePercent: '伤害比例',
  productionBonusRate: '资源产量提升',
  maxAffectedRate: '最大影响比例',
  maxAffectedCount: '最大影响数量',
  minMarchSeconds: '最短行军秒数',
  baseChance: '基础触发率',
  chancePerRatio: '兵力差触发加成',
  maxChance: '最高触发率',
  baseSuppressRate: '基础震慑比例',
  suppressPerRatio: '兵力差震慑加成',
  maxSuppressRate: '最高震慑比例',
}

const OUTCOME_LABELS: Record<string, string> = {
  totalCaptured: '俘虏',
  capturedUnits: '俘虏明细',
  totalRevived: '复活',
  revivedUnits: '复活明细',
  returnedUnits: '返还兵力',
  extraDamage: '额外伤害',
  extraLosses: '追加损失',
  targetExtraLosses: '目标兵种追加损失',
  reducedLosses: '减少损失',
  damagePercent: '伤害比例',
  foodRatio: '口粮比',
  triggerChance: '触发概率',
  effectRate: '效果比例',
  suppressRate: '震慑比例',
  totalSuppressed: '震慑兵力',
  suppressedUnits: '震慑明细',
  preBattleAffected: '战前影响兵力',
  modifiedUnits: '攻防修正',
  disabledTraits: '压制特性',
  disabledTraitCount: '压制特性数',
  marchSeconds: '行军时间',
  beforeSeconds: '原行军秒数',
  afterSeconds: '调整后秒数',
  costReduced: '征兵消耗降低',
  plunderDelta: '掠夺收益变化',
}

export const TRAIT_SCOPE_LABELS: Record<string, string> = {
  self_army: '自己的参战队伍',
  enemy_army: '敌方参战队伍',
  all_army: '全体参战队伍',
  reinforcement_self: '自己的增援队伍',
  defense_self: '守城队伍',
  attack_self: '出征队伍',
}

export const TRAIT_TARGET_LABELS: Record<string, string> = {
  infantry: '步兵',
  cavalry: '骑兵',
  archer: '弓兵',
  special: '特殊兵',
  huWei: '虎卫',
  huBaoQi: '虎豹骑',
  baWangQi: '霸王骑',
}

const VALUE_LABELS: Record<string, string> = {
  attacker: '进攻方',
  defender: '防守方',
  reinforcement: '增援方',
  special: '特殊特性',
  bonus: '加成特性',
  ...TRAIT_SCOPE_LABELS,
  ...TRAIT_TARGET_LABELS,
}

export function formatParamLabel(key: string): string {
  return PARAM_LABELS[key] ?? key
}

export function formatParamValue(key: string, value: number): string {
  if (key.endsWith('Rate') || key.endsWith('Chance') || key.endsWith('Percent') || key === 'effectRate') {
    return `${Math.round(value * 100)}%`
  }
  return value.toLocaleString()
}

export function formatTraitScope(value?: string): string {
  if (!value) return '默认'
  return TRAIT_SCOPE_LABELS[value] ?? value
}

export function formatTraitTarget(value?: string): string {
  if (!value) return '不限'
  return TRAIT_TARGET_LABELS[value] ?? value
}

export function formatTraitOutcomeDetail(key: string, value: TraitOutcomeValue, options: TraitOutcomeFormatOptions = {}): string {
  const label = OUTCOME_LABELS[key] ?? PARAM_LABELS[key] ?? key
  if (typeof value === 'number') {
    if (key.endsWith('Percent') || key.endsWith('Rate') || key.endsWith('Chance') || key === 'effectRate') {
      return `${label}: ${Math.round(value * 100)}%`
    }
    if (key.endsWith('Seconds')) {
      return `${label}: ${formatSeconds(value)}`
    }
    return `${label}: ${value.toLocaleString()}`
  }
  if (typeof value === 'object' && value !== null) {
    return `${label}: ${formatTraitUnitMap(value, options)}`
  }
  return `${label}: ${VALUE_LABELS[value] ?? value}`
}

function formatTraitUnitMap(value: Record<string, number>, options: TraitOutcomeFormatOptions): string {
  const entries = options.sortUnitEntries
    ? options.sortUnitEntries(value, options.faction, options.units)
    : Object.entries(value)
  const text = entries
    .filter(([, amount]) => amount !== 0)
    .map(([unitType, amount]) => {
      const sign = amount > 0 ? '+' : ''
      return `${formatTraitUnitName(unitType, options)} ${sign}${amount.toLocaleString()}`
    })
    .join('、')
  return text || '无'
}

function formatTraitUnitName(unitType: string, options: TraitOutcomeFormatOptions): string {
  if (TRAIT_TARGET_LABELS[unitType]) return TRAIT_TARGET_LABELS[unitType]
  for (const factionUnits of Object.values(options.units ?? {})) {
    if (factionUnits[unitType]?.name) return factionUnits[unitType].name
  }
  return unitType
}

function formatSeconds(value: number): string {
  if (value < 60) return `${value.toLocaleString()} 秒`
  const minutes = Math.floor(value / 60)
  const seconds = value % 60
  if (minutes < 60) return seconds > 0 ? `${minutes} 分 ${seconds} 秒` : `${minutes} 分`
  const hours = Math.floor(minutes / 60)
  const restMinutes = minutes % 60
  return restMinutes > 0 ? `${hours} 小时 ${restMinutes} 分` : `${hours} 小时`
}
