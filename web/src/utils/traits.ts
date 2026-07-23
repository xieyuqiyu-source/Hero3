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
  weiwu_haoling: trait('weiwu_haoling', '魏武号令', 'special', '曹操留城时每分钟自动获得 300 虎卫，不设产兵上限；离城期间停止，后端按真实经过时间权威结算，前端只投影显示，不作为战斗触发特性。', '留城持续生效'),
  weiwu_tongyu: trait('weiwu_tongyu', '魏武统御', 'bonus', '曹操所率全军防御提升 15%，仅在守城或作为援军战斗前生效，主动进攻无效。', '守城/增援战斗前'),
  yibing_touxi: trait('yibing_touxi', '疑兵偷袭', 'special', '进攻、防守或作为援军战斗前有 35% 概率使敌方全军 35% 原始出战兵力直接形成真实伤亡，剩余兵力再计算攻防；概率和比例可由 GM 配置。', '进攻/防守/增援战斗前'),
  mouding_houfa: trait('mouding_houfa', '谋定后发', 'bonus', '防守或作为援军战斗前有 35% 概率使所率全军步兵、骑兵防御提升 35%，主动进攻无效；概率和比例可由 GM 配置。', '防守/增援战斗前'),
  meiren: trait('meiren', '美人心计', 'special', '仅在主动进攻战斗前有 50% 概率使所率全军攻击提升 25%，概率可由 GM 配置。', '主动进攻战斗前'),
  meihuo_raozhen: trait('meihuo_raozhen', '魅惑扰阵', 'bonus', '仅在主动进攻战斗前有 50% 概率使敌方全军步兵、骑兵防御降低 25%，概率可由 GM 配置。', '主动进攻战斗前'),
  huchi_chongzhen: trait('huchi_chongzhen', '虎痴冲阵', 'special', '仅在主动进攻战斗前有 50% 概率使敌方全军步兵、骑兵防御降低 30%；概率和比例可由 GM 配置。', '主动进攻战斗前'),
  huhu_shengwei: trait('huhu_shengwei', '虎虎生威', 'bonus', '被动使许褚所率虎豹骑固定增加 12 点攻击和 5 点移动；进攻、防守、增援均持续生效，不作为战斗触发特性。', '永久被动'),
  huzhu_xuezhan: trait('huzhu_xuezhan', '护主血战', 'special', '防守或增援战斗前必定使典韦所率禁卫甲士固定增加 20 点步兵防御和 20 点骑兵防御；主动进攻无效，固定值可由 GM 配置。', '防守/增援战斗前'),
  sizhandaodi: trait('sizhandaodi', '死战到底', 'bonus', '仅在主动进攻战斗前有 60% 概率使典韦所率步兵攻击提升 35%；概率和比例可由 GM 配置。', '主动进攻战斗前'),
  jixing_benxi: trait('jixing_benxi', '疾行奔袭', 'special', '被动使夏侯渊所率骁骑营固定增加 18 点攻击和 5 点移动；进攻、防守、增援均持续生效，不作为战斗触发特性。', '永久被动'),
  dunzhen_fangyu: trait('dunzhen_fangyu', '盾阵防御', 'bonus', '仅在防守或作为援军战斗前，有 60% 概率使所率全军步兵、骑兵防御提升 30%；概率和比例可由 GM 配置。', '防守/增援战斗前'),
  weizhen_zhenhe: trait('weizhen_zhenhe', '震慑全军', 'special', '仅在主动进攻战斗前有 35% 概率使敌方 25% 兵力溃逃；溃逃兵不参与本次攻防、不计死亡，战后完整返回敌方部队。', '主动进攻战斗前'),
  weizhen_xiaoyao: trait('weizhen_xiaoyao', '威震逍遥', 'bonus', '仅在主动进攻战斗前有 60% 概率使所带骑兵攻击提升 35%。', '主动进攻战斗前'),
  shengui_zhicai: trait('shengui_zhicai', '神鬼之才', 'special', '全天候永久使郭嘉固定增加 10 点内政和 10 点智谋；进入最终四维及对应属性，不作为战斗触发特性。', '永久被动'),
  guicai_yice: trait('guicai_yice', '鬼才遗策', 'bonus', '进攻、守城或作为援军战斗结束后，默认必定按郭嘉所率部队本场真实阵亡逐兵种复活 22%，返回对应部队；比例和概率可由 GM 配置。', '进攻/防守/增援战斗结束后'),
  wangzuo_zhicai: trait('wangzuo_zhicai', '王佐之才', 'special', '荀彧留城时降低 5% 征兵资源消耗，离城失效，不作为战斗触发特性。', '留城征兵消耗时'),
  neizheng_jingying: trait('neizheng_jingying', '内政精营', 'bonus', '荀彧作为主将留城时被动提升 5% 资源产量，离城失效，不作为战斗触发特性。', '留城被动生效'),
  rende: trait('rende', '仁德天下', 'special', '全天候永久使刘备固定增加 10 点内政和 12 点统率；进入最终四维及对应属性，不作为战斗触发特性。', '永久被动'),
  renzhu_shouhu: trait('renzhu_shouhu', '仁主守护', 'bonus', '进攻、守城或作为援军战斗结束后，有 60% 概率按刘备所率部队本场真实阵亡逐兵种复活 35%，返回对应部队；比例和概率可由 GM 配置。', '进攻/防守/增援战斗结束后'),
  shuiyan_qijun: trait('shuiyan_qijun', '水淹七军', 'special', '仅在主动进攻战斗前有 35% 概率使敌方全军 30% 原始出战兵力直接形成真实伤亡，剩余兵力再计算攻防。', '主动进攻战斗前'),
  wusheng_pojun: trait('wusheng_pojun', '武圣破军', 'bonus', '仅在主动进攻战斗前有 50% 概率使关羽所率青龙军攻击提升 38%。', '主动进攻战斗前'),
  zhenhe_quanjun: trait('zhenhe_quanjun', '万人怒吼', 'special', '仅在主动进攻战斗前有 50% 概率使敌方全军 50% 兵力溃逃；溃逃兵不参与本次攻防、不计死亡，战后完整返回。', '主动进攻战斗前'),
  wanren_nuhou: trait('wanren_nuhou', '勇冠三军', 'bonus', '仅在主动进攻战斗前有 40% 概率使张飞所率南蛮象攻击提升 35%。', '主动进攻战斗前'),
  qimen_dunjia: trait('qimen_dunjia', '奇门遁甲', 'special', '进攻、防守或作为援军战斗前，使敌方 25% 兵力仅本场无法参战且战后完整保留；比例和概率可由 GM 配置。', '进攻/防守/增援战斗前'),
  wolong_mouzhi: trait('wolong_mouzhi', '卧龙奇谋', 'bonus', '进攻、防守或作为援军战斗前，有 60% 概率封禁敌方所有参战将领的战斗触发型特性，不影响永久被动；双方都有诸葛亮时该特性均失效。', '进攻/防守/增援战斗前'),
  longdan_jiuyuan: trait('longdan_jiuyuan', '龙胆救援', 'special', '防守或增援战斗前固定使赵云所率麒麟卫步防、骑防各提升 25%；掠夺结算时守城主将保护 20% 资源，赵云援军按 20%、10%、5% 依次递减叠加。', '防守/增援战斗前及掠夺结算'),
  qijin_qichu: trait('qijin_qichu', '七进七出', 'bonus', '主动出征或增援创建时固定提升 100% 行军速度，最低 60 秒，不作为战斗触发特性。', '行军创建时'),
  xiliang_tuji: trait('xiliang_tuji', '西凉突击', 'special', '进攻、守城或作为援军时有 35% 概率按敌方骑兵战前人数尝试追加 12% 损失，实际扣兵以战报为准。', '战斗结算后'),
  tianshen_xiafan: trait('tianshen_xiafan', '天神下凡', 'bonus', '被动增加 20 点武力，每点武力转化为 2% 部队攻击。', '被动属性'),
  baibu_chuanyang: trait('baibu_chuanyang', '百步穿杨', 'special', '仅在主动进攻战斗前有 45% 概率使敌方全军步兵、骑兵防御降低 30%。', '主动进攻战斗前'),
  laodang_yizhuang: trait('laodang_yizhuang', '老当益壮', 'bonus', '永久使黄忠固定增加 12 点武力和 12 点统率；进入最终四维及对应攻防属性，不作为战斗触发特性。', '永久被动'),
  qibing_raohou: trait('qibing_raohou', '奇兵绕后', 'special', '被动使魏延所率南蛮象固定增加 18 点攻击和 15 点移动；进攻、防守、增援均持续生效，不作为战斗触发特性。', '永久被动'),
  gushou_hanzhong: trait('gushou_hanzhong', '固守汉中', 'bonus', '仅在防守或作为援军时，使所带部队的步兵、骑兵防御各增加 20 点。', '防守/增援战斗前'),
  jiangdong_haoling: trait('jiangdong_haoling', '江东号令', 'special', '仅在掠夺战防守失败时使敌方最终掠夺收益降低 20%，普通进攻或防守获胜时无效。', '掠夺结算时'),
  jiangdong_gushou: trait('jiangdong_gushou', '江东固守', 'bonus', '仅在防守或作为援军时，有 50% 概率使所带部队的步兵、骑兵防御提升 50%。', '防守/增援战斗前'),
  xiaobawang_zhuiji: trait('xiaobawang_zhuiji', '小霸王追击', 'special', '仅在掠夺战胜利后有 35% 概率按敌方各兵种战前人数尝试追加 10% 损失，实际扣兵以战报为准。', '掠夺战结算后'),
  xiaobawang_tieqi: trait('xiaobawang_tieqi', '小霸王', 'bonus', '仅在主动进攻时使所带霸王骑的单位攻击固定增加 50 点。', '主动进攻战斗前'),
  huogong: trait('huogong', '火烧赤壁', 'special', '仅在主动进攻战斗结算后必定触发，按敌方各兵种战前人数尝试追加 25% 损失，实际扣兵以战报为准。', '主动进攻战斗结算后'),
  meizhoulang_junlue: trait('meizhoulang_junlue', '美周郎军略', 'bonus', '仅在主动进攻时使所带全军攻击提升 5%。', '主动进攻战斗前'),
  huoshao_lianying: trait('huoshao_lianying', '火烧联营', 'special', '进攻、守城或作为援军时，战斗结算后有 35% 概率按敌方步兵战前人数追加 100% 损失，使目标步兵最终全灭。', '战斗结算后'),
  lianying_zengshang: trait('lianying_zengshang', '连营增伤', 'bonus', '进攻、守城或作为援军时，战斗结算后按敌方步兵战前人数尝试追加 10% 损失，实际扣兵以战报为准。', '战斗结算后'),
  baiyi_dujiang: trait('baiyi_dujiang', '白衣渡江', 'special', '主动出征或增援创建时有 35% 概率提升 20% 行军速度，最低 60 秒，不作为战斗触发特性。', '行军创建时'),
  baiyi_jixing: trait('baiyi_jixing', '白衣急行', 'bonus', '主动出征或增援创建时固定提升 20% 行军速度，最低 60 秒，可与白衣渡江逐次叠加。', '行军创建时'),
  kuairu_shandian: trait('kuairu_shandian', '快如闪电', 'special', '主动出征或增援创建时有 35% 概率提升 400% 行军速度，最低 30 秒，不作为战斗触发特性。', '行军创建时'),
  xinyi_yonglie: trait('xinyi_yonglie', '信义勇烈', 'bonus', '只提升将领所带援军自身 10% 步兵、骑兵防御，主动进攻或主城守军无效。', '援军战斗前'),
  jinfan_jielue: trait('jinfan_jielue', '锦帆劫掠', 'special', '仅在掠夺战主动进攻获胜时使最终掠夺收益提升 20%，普通进攻或掠夺战败时无效。', '掠夺结算时'),
  jinfan_qixi: trait('jinfan_qixi', '锦帆奇袭', 'bonus', '仅在掠夺战主动进攻时使所带全军攻击提升 10%，普通进攻无效。', '掠夺战战斗前'),
  kurouji: trait('kurouji', '苦肉计', 'special', '战斗结算后有 35% 概率拦截敌方 1 项后续特性；没有可拦截特性时实际压制数为 0。', '战斗结算后'),
  kurou_fanji: trait('kurou_fanji', '苦肉反击', 'bonus', '进攻、守城或作为援军时，战斗结算后按敌方各兵种战前人数尝试追加 10% 损失，实际扣兵以战报为准。', '战斗结算后'),
}

export const GENERAL_TRAITS: Record<string, string[]> = {
  caocao: ['weiwu_haoling', 'weiwu_tongyu'],
  simayi: ['yibing_touxi', 'mouding_houfa'],
  zhenmi: ['meiren', 'meihuo_raozhen'],
  xuchu: ['huchi_chongzhen', 'huhu_shengwei'],
  dianwei: ['huzhu_xuezhan', 'sizhandaodi'],
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
  forceBonus: '武力增加',
  intelligenceBonus: '智谋增加',
  politicsBonus: '内政增加',
  commandBonus: '统率增加',
  generalDefenseFlat: '全军防御增加',
  unitAttackFlat: '兵种攻击',
  unitSpeedFlat: '兵种移动',
  enemyDefenseReductionRate: '敌方防御降低',
  lossReductionRate: '损失降低',
  reviveRate: '复活比例',
  maxReviveCount: '复活上限',
  returnRate: '返还比例',
  maxReturnCount: '返还上限',
  plunderBonusRate: '掠夺修正',
  plunderProtectionRate: '资源保护比例',
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
	triggerCount: '触发场次',
	totalCaptured: '俘虏总数',
  capturedUnits: '俘虏归队',
  capturedToGarrison: '俘虏驻防',
  totalRevived: '复活总数',
  revivedUnits: '复活兵力',
  actualLostUnits: '本场真实阵亡',
  returnedUnits: '返还兵力',
  returnedFledUnits: '战后返回兵力',
  extraDamage: '额外伤害',
  extraLosses: '追加损失',
  targetExtraLosses: '目标兵种追加损失',
  reducedLosses: '减少损失',
  damagePercent: '设计伤害比例',
  foodRatio: '口粮比',
  triggerChance: '触发概率',
  effectRate: '设计效果比例',
  captureMax: '设计单兵种俘虏上限',
  maxAffectedRate: '设计最大影响比例',
  suppressRate: '震慑比例',
  totalSuppressed: '震慑兵力',
  suppressedUnits: '本场压制兵力',
  fledUnits: '本场溃逃兵力',
  preBattleAffected: '战前真实伤亡',
  realCasualties: '真实伤亡兵力',
  totalRealCasualties: '真实伤亡总数',
  modifiedUnits: '实际攻防修正',
  attackModifiedUnits: '实际攻击修正',
  attackBonusRate: '设计攻击加成',
  unitAttackFlat: '设计单位攻击增加',
  unitSpeedFlat: '设计单位移动增加',
  attackReductionRate: '设计攻击降低',
  enemyDefenseReductionRate: '设计敌方防御降低',
  defenseBonusRate: '设计防御加成',
  generalDefenseFlat: '设计全军防御增加',
  lossReductionRate: '设计减损比例',
  maxReviveCount: '设计复活上限',
  maxReturnCount: '设计返还上限',
  plunderBonusRate: '设计掠夺修正',
  infantryDefenseModifiedUnits: '实际步防修正',
  cavalryDefenseModifiedUnits: '实际骑防修正',
  disabledTraits: '压制特性',
  disableTraitCount: '设计压制特性数',
  disabledGeneralCount: '封禁将领数',
  disabledTraitCount: '实际压制特性数',
  status: '状态',
  invalidReason: '失效原因',
  marchSeconds: '行军时间',
  beforeSeconds: '原行军秒数',
  afterSeconds: '调整后秒数',
  costReduced: '征兵消耗降低',
  plunderDelta: '掠夺资源修正',
  plunderProtectionContributionRate: '本次资源保护',
  cumulativePlunderProtectionRate: '累计资源保护',
  protectedResources: '实际保护资源',
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
  qingZhouArmy: '青州军',
  jinWeiSoldier: '禁卫甲士',
  huWei: '虎卫',
  zhanYingTanMa: '战鹰探马',
  qiQiYing: '骁骑营',
  huBaoQi: '虎豹骑',
  chongZhuangChe: '冲撞车',
  luLeiChe: '露雷车',
  jianzhuShi: '建筑师',
  tuZu: '士族',
  weiMerchant: '魏国商人',
  azureDragon: '青龙军',
  flyingKite: '飞鸢',
  greedyWolf: '贪狼营',
  hanRoyalty: '汉室宗亲',
  qilinGuard: '麒麟卫',
  shuMerchant: '蜀国商人',
  siegeTower: '临冲车',
  southernElephant: '南蛮象',
  thunderBolt: '霹天雷',
  woodenOx: '木牛流马',
  xiLiangCavalry: '西凉铁骑',
  shadowGuard: '影卫',
  xiuLuo: '修罗',
  secretAgent: '密探',
  divineWind: '神风',
  zhuQueRider: '朱雀骑',
  baWangQi: '霸王骑',
  overlordRider: '霸王骑',
  chongChe: '冲撞车',
  juShiChe: '巨石车',
  fengShuiMaster: '风水师',
  taiPingShi: '太平士',
  wuMerchant: '吴国商人',
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

const TRAIT_RESOURCE_LABELS: Record<string, string> = {
  wood: '木材',
  stone: '石料',
  iron: '铁矿',
  food: '粮食',
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

// formatTraitOutcomeDetails 将单次触发的全部设计参数和实际结果汇总为战报文本。
export function formatTraitOutcomeDetails(detail?: Record<string, TraitOutcomeValue>, options: TraitOutcomeFormatOptions = {}): string {
  if (!detail) return ''
  return Object.entries(detail)
    .filter(([key]) => key !== 'suppressedUnits' || detail.fledUnits == null)
    .map(([key, value]) => formatTraitOutcomeDetail(key === 'returnedUnits' && detail.fledUnits != null ? 'returnedFledUnits' : key, value, options))
    .filter(Boolean)
    .join('；')
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
  if (TRAIT_RESOURCE_LABELS[unitType]) return TRAIT_RESOURCE_LABELS[unitType]
  if (TRAIT_TARGET_LABELS[unitType]) return TRAIT_TARGET_LABELS[unitType]
  if (OUTCOME_LABELS[unitType]) return OUTCOME_LABELS[unitType]
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
