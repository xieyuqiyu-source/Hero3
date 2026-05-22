export interface ApiDocItem {
  id: string
  method: 'GET' | 'POST'
  path: string
  title: string
  desc: string
  usedBy: string
  status: '已实现' | '待实现'
  sampleBody?: unknown
}

export const apiDocs: ApiDocItem[] = [
  {
    id: 'health',
    method: 'GET',
    path: '/healthz',
    title: '健康检查',
    desc: '确认 Go 后端是否启动。',
    usedBy: 'admin 顶部环境状态',
    status: '已实现',
  },
  {
    id: 'game-state',
    method: 'GET',
    path: '/api/v1/game/state',
    title: '游戏状态',
    desc: '读取当前玩家主界面状态，不传 playerId 时返回 demo 状态。',
    usedBy: 'web 主界面、admin 玩家状态概览',
    status: '已实现',
  },
  {
    id: 'register',
    method: 'POST',
    path: '/api/v1/accounts/register',
    title: '注册账号',
    desc: '创建轻账号，返回 accountId 和 username。',
    usedBy: 'web 云同步弹窗',
    status: '已实现',
    sampleBody: { username: 'api_check_user', password: '123456' },
  },
  {
    id: 'login',
    method: 'POST',
    path: '/api/v1/accounts/login',
    title: '登录账号',
    desc: '校验账号密码，返回 accountId 和 username。',
    usedBy: 'web 云同步弹窗',
    status: '已实现',
    sampleBody: { username: 'xieyuqi', password: '123456' },
  },
  {
    id: 'account-players',
    method: 'GET',
    path: '/api/v1/accounts/acc_958538e3bc17faf00407ddcc/players',
    title: '账号存档列表',
    desc: '读取指定账号绑定的所有 playerId。',
    usedBy: 'web 云同步存档选择',
    status: '已实现',
  },
  {
    id: 'create-player',
    method: 'POST',
    path: '/api/v1/players/create',
    title: '创建存档',
    desc: '为账号创建新的游戏存档。',
    usedBy: 'web 选择阵营后进入游戏',
    status: '已实现',
    sampleBody: {
      accountId: 'acc_958538e3bc17faf00407ddcc',
      nickname: '接口测试主公',
      faction: 'wei',
    },
  },
  {
    id: 'admin-accounts',
    method: 'GET',
    path: '/api/v1/admin/accounts',
    title: 'GM 账号与存档',
    desc: '读取所有注册账号和每个账号下的存档。',
    usedBy: 'admin 注册玩家与存档列表',
    status: '已实现',
  },
]
