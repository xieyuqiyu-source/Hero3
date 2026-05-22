export interface ApiDocItem {
  id: string
  method: 'GET' | 'POST' | 'PUT' | 'DELETE'
  path: string
  title: string
  desc: string
  usedBy: string
  status: '已实现' | '待实现'
  destructive?: boolean
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
    id: 'delete-account',
    method: 'DELETE',
    path: '/api/v1/accounts/acc_958538e3bc17faf00407ddcc',
    title: '删除账号',
    desc: '删除账号，并删除该账号关联的全部云存档。',
    usedBy: 'admin 注册玩家与存档列表',
    status: '已实现',
    destructive: true,
  },
  {
    id: 'delete-player',
    method: 'DELETE',
    path: '/api/v1/players/player_47f23e0eeb08ab50402fedfc',
    title: '删除云存档',
    desc: '删除指定 playerId 对应的云存档，不删除账号。',
    usedBy: 'admin 注册玩家与存档列表',
    status: '已实现',
    destructive: true,
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
  {
    id: 'admin-balance-get',
    method: 'GET',
    path: '/api/v1/admin/balance',
    title: 'GM 读取数值配置',
    desc: '读取资源基础产量、建筑每级产量、仓库容量、升级消耗和升级时间。',
    usedBy: 'admin 游戏数值配置',
    status: '已实现',
  },
  {
    id: 'admin-balance-update',
    method: 'PUT',
    path: '/api/v1/admin/balance',
    title: 'GM 保存数值配置',
    desc: '保存 GM 修改后的资源与建筑数值配置。',
    usedBy: 'admin 游戏数值配置',
    status: '已实现',
    destructive: true,
  },
]
