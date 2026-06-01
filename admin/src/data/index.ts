export const resourceActions = [
  { title: '补发资源', desc: '待权限、审计、二次确认接入后启用' },
  { title: '清理队列', desc: '待队列模型和操作日志接入后启用' },
  { title: '重置地图', desc: '待地图刷新规则接入后启用' },
]

export const systemActions = [
  { title: '发布维护公告', level: 'normal' },
  { title: '刷新配置缓存', level: 'normal' },
  { title: '冻结玩家操作', level: 'danger' },
]

export const auditLogs = [
  { time: '15:20', action: 'admin 调整 player_47f23e0e 粮食 +5000' },
  { time: '15:06', action: 'system 生成测试地图目标' },
  { time: '14:48', action: 'admin 刷新战斗规则配置' },
]

export const guardrails = [
  { title: '权限隔离', desc: '后续所有高危操作必须接入管理员身份和二次确认。' },
  { title: '审计优先', desc: '资源调整、封禁、配置刷新必须写入不可篡改日志。' },
  { title: '测试服优先', desc: '当前后台只面向开发环境，不连接正式服数据。' },
]
