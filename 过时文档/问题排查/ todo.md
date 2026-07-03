# 问题排查 TODO

> 上轮已完成：货币接口所有权边界（admin token + requireOwnership 双层鉴权）、建筑快速完成 UI（资源/非资源建筑均支持点击倒计时）。

## High

### 货币变化没有独立流水，只有当前余额
- **位置**: `go/internal/game/service_gold.go::AddGold/DeductGold`（`_ = reason // TODO: 记录流水日志`）, `go/internal/storage/mysql.go`
- **问题**: 只存 accounts.gold 和 players.state_json.cityGold，reason 直接丢掉，没有不可变流水表
- **影响**: 补偿、封禁追查、充值对账、异常回滚、客服申诉都无法定位"钱是怎么来的、怎么没的"
- **建议**: 建 gold_ledger 流水表（时间、账户/玩家ID、类型、金额、余额快照、reason），每次变动写一条记录，GM 后台加查询入口

## Medium

### 兑换缺少幂等控制（实际风险已被冷却挡住，但仍未实现）
- **位置**: `go/internal/game/service_gold.go::ExchangeGoldToCityGold/ExchangeCityGoldToGold`, `go/internal/api/handlers.go::ExchangeGold/ReverseExchangeGold`
- **现状**: `getPlayerLock` 串行化 + `LastExchangeAt` + 1 小时 `ExchangeCooldownSecs` 冷却，已能挡住双击 / 网络抖动重复扣加；但仍未实现真正的幂等键
- **影响**: 当前兑换接口实战风险低；隐患在于未来如果新增"无冷却的货币写接口"（如商店购买、礼包兑换、活动消耗），需要再单独设计幂等
- **建议**: 抽出一个统一的"货币写操作"中间层，强制要求请求带 idempotency key，库里建 `idempotency_keys` 去重表（过期清理）；后续所有货币写操作复用

## 待实现

### 内政 / 防御功能建筑正式接入
- **候选建筑**:
  - 建造司：提升建筑速度，预期满级约 ×10 建筑速度
  - 内政厅：提升资源产量，预期满级约 ×16 产量
  - 驿站：提升行军速度，预期满级约 ×5 行军速度
  - 城墙：提升城池防御，预期每级 +5% 防御
  - 烽火台：预警 / 侦查来犯敌军，具体效果等来袭/侦查系统读取等级
- **暂不提交半成品原因**:
  - 前端军事建筑页已经有展示入口，但线上 `balance.json` 未配置这些建筑时，默认加入存档会造成可见但不可升级/效果不完整。
  - 烽火台、城墙等还依赖后续 PvP / 来袭 / 驻防逻辑，目前只加配置会让玩家误以为功能已完成。
- **正式开发时需要一起补齐**:
  - `go/config/balance.json` 与 `defaultBalance` 同步配置
  - 新玩家默认建筑、旧存档迁移
  - 前端展示文案和锁定状态
  - 对应数值测试
  - 城墙/烽火台接入 PvP、侦查或来袭系统后再开放

### 极速征兵费用计算 —— 设计方向需确认 ❓
- **当前代码**: `service_recruit.go::InstantCompleteRecruit` 使用 `endsAt - now` 作为剩余秒数（**包含**排队等待时间），前端 `RecruitQueuePanel.tsx::getInstantCost` 同步使用 `getRemainingSeconds(queue.endsAt)`
- **原 TODO 主张**: 应该使用 `unitConfig.TrainSeconds × amount`（**不包含**排队等待时间）
- **冲突点**: 多个队列时，越往后的队列剩余秒数越大，城金费用越高
  - **方案 A（当前）**: 玩家为"立刻拿到这批兵"付费，等待时间也算进去
  - **方案 B（原 TODO）**: 玩家只为"这批兵的训练时长"付费，等待时间免费
- **建议**: 先和策划/产品确认意图，再决定是否回滚到方案 B；当前实现至少前后端一致，没有数据不一致问题

### 闪光城池掉落金币系统
- **概念**: 任何等级的 NPC 城池都有概率带"闪光"词条，打赢闪光城池掉落金币（账户级）
- **闪光概率**: 10%（1/10），GM 可调（`npc.json` 加 `shinyRate` 字段）
- **掉落数量（按城池等级）**:
  - 小型: 1~10 金币
  - 中型: 10~50 金币
  - 大型: 50~100 金币
  - 金色: 100~200 金币
  - GM 可调（`npc.json` 加 `shinyGoldDrop` 按等级配置 min/max）
- **金币归属**: 加到玩家的账户金币（Account.Gold），不是城金；可复用现成的 `repo.AddAccountGold(accountId, amount)`
- **缺什么**:
  - `NpcCity` 加 `shiny bool` 字段，生成时按 `shinyRate` 概率标记
  - `BattleReport` 加 `goldDrop int` 字段
  - `repository` 加 `GetAccountIDByPlayerID(playerID) (string, error)` 方法
  - `AttackNpc` 战斗胜利分支检查 shiny → 计算金额 → AddAccountGold
- **未登录玩家处理**:
  - 通过 playerId 反查 accountId
  - 如果查不到账户（本地玩家），在攻击闪光城池前弹窗提醒："掉落稀有道具，是否创建账号同步？"
  - 玩家可选择去登录/注册，或跳过（跳过则金币丢失）
- **前端展示**:
  - 城池卡片加闪光 CSS 动画特效（一眼可辨）
  - 战报里显示金币掉落数量
  - 战斗结果弹窗高亮显示金币奖励
