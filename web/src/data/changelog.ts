/**
 * 更新日志
 * 每次发版在顶部添加一条，version 递增
 * 前端通过对比 localStorage 中存储的已读版本号决定是否弹窗
 */

export interface ChangelogEntry {
  version: number
  date: string
  title: string
  items: string[]
}

export const changelog: ChangelogEntry[] = [
  {
    version: 5,
    date: '2026-05-23',
    title: '征兵系统上线',
    items: [
      '军事页面全新改版：征兵 / 将领 / 科技三大模块',
      '按阵营展示专属兵种，步兵/骑兵/攻城/特殊四分类',
      '征兵弹窗：属性面板、资源消耗、数量调节、一键征满',
      '征兵队列：串行训练、实时倒计时、完成通知',
      '侧边栏军队区域显示实际兵种和数量',
      '顶部资源栏全页面显示',
    ],
  },
  {
    version: 4,
    date: '2026-05-22',
    title: '城建系统扩展',
    items: [
      '新增功能建筑：建造司、步兵营、骑兵营、兵器司、防具司、内政厅、粮仓、驿站、城墙、烽火台、集市',
      '城建 tab 分组展示：军事 / 内政 / 防御',
      '更新公告弹窗上线，支持版本号对比自动弹出',
    ],
  },
  {
    version: 3,
    date: '2026-05-22',
    title: '建筑升级系统上线',
    items: [
      '资源建筑支持单独升级和一键升级',
      '仓库、军事建筑支持升级',
      '升级倒计时实时显示',
      '新增一键爆仓功能（测试用）',
      '新增产量加成入口（功能开发中）',
      'API 错误消息中文化',
    ],
  },
  {
    version: 1,
    date: '2026-05-21',
    title: '初始版本',
    items: [
      '三国阵营选择与角色创建',
      '云同步登录与存档管理',
      '城池资源建筑展示',
      '资源实时增长预测',
      '深色/亮色主题切换',
    ],
  },
]

/** 当前最新版本号 */
export const LATEST_VERSION = changelog[0].version
