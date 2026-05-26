# 问题排查 TODO

## Critical

### 货币接口没有所有权边界
- **位置**: `go/internal/api/handlers.go` (AddGold, DeductGold, ExchangeGold, ReverseExchangeGold)
- **问题**: 所有接口直接信任客户端传来的 playerId/accountId，没有 token/session，也没有归属校验
- **影响**: 货币系统可以被越权操作，别人账号的金币和城金理论上都能被改
- **建议**: 登录签发 JWT，加 auth 中间件，校验 playerId 归属，AddGold/DeductGold 限 admin-only

## High

### 货币变化没有独立流水，只有当前余额
- **位置**: `go/internal/game/service_gold.go` (line 23, 44, 68), `go/internal/storage/mysql.go` (line 43, 87)
- **问题**: 只存 accounts.gold 和 players.state_json.cityGold，reason 直接丢掉，没有不可变流水表
- **影响**: 补偿、封禁追查、充值对账、异常回滚、客服申诉都无法定位"钱是怎么来的、怎么没的"
- **建议**: 建 gold_ledger 流水表（时间、账户/玩家ID、类型、金额、余额快照、reason），每次变动写一条记录，GM 后台加查询入口


## Medium

### 兑换缺少幂等控制，重复提交会重复扣加
- **位置**: `go/internal/game/service_gold.go` (line 87, 131), `go/internal/api/handlers.go` (line 672)
- **问题**: 兑换接口没有请求幂等键，客户端重试、双击、网络抖动都可能触发两次
- **影响**: 金币系统会出现"同一笔操作被算两次"的投诉，排查成本很高
- **建议**: 给货币写操作加请求唯一 ID（idempotency key），在库里做去重；至少兑换类接口先补这一层


## 待实现

### 极速征兵费用计算问题
- **问题**: 多个队列时，越往后的队列显示的城金花费越高（因为 endsAt 包含了前面队列的等待时间），但实际极速完成的只是选中的那一个队列本身的训练时间，不应该把排队等待时间算进去
- **建议**: 计算费用时应该用队列自身的训练时长（unitConfig.TrainSeconds × amount），而不是 endsAt 减去当前时间

### 建筑快速完成 UI
- **需求**: 任何建筑只要在升级中（显示了倒计时），就可以点击时间直接触发快速完成（城金消费 + 二次确认弹窗）
- **范围**: 资源建筑可以一键全部完成，也可以单个点击时间完成；非资源建筑（仓库等）也支持点击时间完成
