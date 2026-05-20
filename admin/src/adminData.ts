export const overviewStats = [
  { label: '在线玩家', value: '128', hint: '近 5 分钟活跃' },
  { label: '今日创建', value: '42', hint: '测试服新增角色' },
  { label: '战斗结算', value: '1,284', hint: '今日战报数量' },
  { label: '异常告警', value: '3', hint: '等待处理' },
]

export const playerRows = [
  { id: 'demo-player', name: '主公', power: '战力 8,420', status: '在线' },
  { id: 'player-wei-021', name: '魏境守备', power: '战力 6,310', status: '离线 12m' },
  { id: 'player-shu-017', name: '蜀道先锋', power: '战力 5,980', status: '征兵中' },
]

export const resourceActions = [
  { title: '补发资源', desc: '按玩家 ID 发放木、石、铁、粮' },
  { title: '清理队列', desc: '处理卡住的升级或征兵任务' },
  { title: '重置地图', desc: '刷新玩家测试地图目标' },
]

export const systemActions = [
  { title: '发布维护公告', level: 'normal' },
  { title: '刷新配置缓存', level: 'normal' },
  { title: '冻结玩家操作', level: 'danger' },
]

export const auditLogs = [
  { time: '15:20', action: 'admin 调整 demo-player 粮食 +5000' },
  { time: '15:06', action: 'system 生成测试地图目标' },
  { time: '14:48', action: 'admin 刷新战斗规则配置' },
]

export const guardrails = [
  { title: '权限隔离', desc: '后续所有高危操作必须接入管理员身份和二次确认。' },
  { title: '审计优先', desc: '资源调整、封禁、配置刷新必须写入不可篡改日志。' },
  { title: '测试服优先', desc: '当前后台只面向开发环境，不连接正式服数据。' },
]
