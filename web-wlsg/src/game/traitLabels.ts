/** 将领特性标签集中映射后端稳定 ID，供战报等玩家页面显示中文名称。 */
const traitLabels: Record<string, string> = {
  weiwu_haoling: '魏武号令', weiwu_tongyu: '魏武统御', yibing_touxi: '疑兵偷袭', mouding_houfa: '谋定后发',
  meiren: '美人心计', meihuo_raozhen: '魅惑扰阵', huchi_chongzhen: '虎痴冲阵', huhu_shengwei: '虎虎生威',
  huzhu_xuezhan: '护主血战', sizhandaodi: '死战到底', jixing_benxi: '疾行奔袭', dunzhen_fangyu: '盾阵防御',
  weizhen_zhenhe: '震慑全军', weizhen_xiaoyao: '威震逍遥', shengui_zhicai: '神鬼之才', guicai_yice: '鬼才遗策',
  wangzuo_zhicai: '王佐之才', neizheng_jingying: '内政精营', rende: '仁德天下', renzhu_shouhu: '仁主守护',
  shuiyan_qijun: '水淹七军', wusheng_pojun: '武圣破军', zhenhe_quanjun: '震慑全军', wanren_nuhou: '万人怒吼',
  qimen_dunjia: '奇门遁甲', wolong_mouzhi: '卧龙奇谋', longdan_jiuyuan: '龙胆救援', qijin_qichu: '七进七出',
  xiliang_tuji: '西凉突击', tianshen_xiafan: '天神下凡', baibu_chuanyang: '百步穿杨', laodang_yizhuang: '老当益壮',
  qibing_raohou: '奇兵绕后', gushou_hanzhong: '固守汉中', jiangdong_haoling: '江东号令', jiangdong_gushou: '江东固守',
  xiaobawang_zhuiji: '小霸王追击', xiaobawang_tieqi: '小霸王', huogong: '火烧赤壁', meizhoulang_junlue: '美周郎军略',
  huoshao_lianying: '火烧联营', lianying_zengshang: '连营增伤', baiyi_dujiang: '白衣渡江', baiyi_jixing: '白衣急行',
  kuairu_shandian: '快如闪电', xinyi_yonglie: '信义勇烈', jinfan_jielue: '锦帆劫掠', jinfan_qixi: '锦帆奇袭',
  kurouji: '苦肉计', kurou_fanji: '苦肉反击',
}

/** 优先使用已知中文名，未知扩展特性保留后端名称或 ID。 */
export function traitLabel(traitId: string, backendName?: string) {
  const name = backendName?.trim()
  return traitLabels[traitId] ?? (name && name !== traitId ? name : traitId)
}
