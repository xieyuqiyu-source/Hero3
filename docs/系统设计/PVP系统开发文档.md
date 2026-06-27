# <span style="color:#dc2626">Hero3 PVP 系统开发文档</span>

最后更新：2026-06-27

## <span style="color:#dc2626">0. 当前第一版实现状态</span>

2026-06-27 已完成第一版 PVP 战斗主链路：

- PVP 模块已登记到玩法模块注册表。
- 已新增 `pvp_marches`、`pvp_battles`、`pvp_player_states`、`pvp_seasons`、`pvp_season_players` 表结构。
- 已新增 PVP Repository 标准接口和 MySQL/内存实现。
- 已实现玩家目标列表、玩家侦查、发起攻击/掠夺行军、我的行军、战斗列表接口。
- 已实现单场 PVP 战斗详情接口 `/api/v1/pvp/battles/{battleId}`。
- 发起攻击会从攻击方长期兵力扣出出征兵力，并占用出征武将。
- 到期行军会进入 PVP 专用结算事务，调用 `internal/core/combat` 的 PVP 场景规则。
- 防守方自有兵力会参与战斗。
- 防守方驻防队伍会作为防守援军参与战斗，援军损耗写回对应驻防/增援记录。
- 攻击方幸存兵力会返还攻击方长期兵力。
- 攻击方、防守方都会生成 PVP 战报。
- 攻击胜利时会按仓库保护量和剩余负重进行基础资源掠夺。
- 地图页“玩家”标签已展示玩家目标，并提供侦查、增援、攻击、掠夺按钮；玩家卡片样式比 NPC 卡片更紧凑。
- 已实现 PVP 行军召回，召回后进入返程状态。
- 返程到期后会返还出征兵力并释放 PVP 武将占用。
- 已实现 PVP 行军城金加速，使用 `CityGoldPerSecond` 计算费用，并记录加速次数和城金流水。
- MySQL 跨玩家事务已按玩家 ID 固定顺序锁定双方玩家状态，降低互相攻击或增援时的死锁风险。
- 已实现 PVP 基础保护期、攻击冷却、每日攻击次数限制。
- 防守方战败后会写入战败保护期。
- 已实现第一版 PVP 积分和胜负统计，当前存放在 `pvp_player_states.metadata_json`。
- 已实现第一版复仇记录，防守方被攻击后生成复仇机会，复仇成功后关闭对应记录。
- 已新增 `/api/v1/pvp/state` 和 `/api/v1/pvp/revenge` 查询接口。
- 已实现独立 PVP 赛季摘要和积分排行榜查询，排行榜当前从 `pvp_player_states.metadata_json` 汇总。
- 已新增 `/api/v1/pvp/season` 和 `/api/v1/pvp/rankings` 查询接口。
- 已实现第一版赛季结算，结算时会将排行榜固化到 `pvp_season_players`。
- 已实现第一版赛季奖励信函，结算时按名次通过系统信函发放城金奖励。
- PVP 战报已补充积分变化、参战援军摘要、援军损耗字段。
- 前端军情详情已展示 PVP 积分变化、参战驻防/援军、援军损耗；战斗结果弹窗已展示 PVP 结算摘要。
- 已新增 GM PVP 只读查询接口和后台页面，可查询玩家 PVP 状态、排行榜、最近行军和最近战斗。
- 已实现第一版防守武将自动出战策略：防守方当前主将自动参战，战斗详情和战报会记录攻防双方参战武将快照。
- 已实现第一版主动免战和系统/维护保护：`pvp_protection` 道具效果可写入 `manual` 保护，GM 后台可设置 `system`/`maintenance` 保护，目标列表和发起攻击都会拦截保护中目标。
- 已实现 GM PVP 修复动作：可强制结算 `marching/resolving` 行军；可取消尚未生成战斗的 `marching/returning/resolving` 行军并返还出征兵力和释放武将。
- 已新增 GM 赛季管理入口：`GET /api/v1/admin/pvp/seasons`、`POST /api/v1/admin/pvp/seasons`、`PUT /api/v1/admin/pvp/seasons/{seasonId}`、`POST /api/v1/admin/pvp/seasons/{seasonId}/settle`。

后续继续开发时，应在当前第一版主链路上继续增强更细奖励档位和自动赛季轮换，不应改回即时战斗。

## <span style="color:#dc2626">1. 文档定位</span>

本文档定义 Hero3 PVP 系统的最终标准设计，用于指导后续完整 PVP 玩法开发。

PVP 系统是玩法模块，不是核心资产系统。它负责玩家之间的攻击、行军、战斗、防守、增援接入、战报、保护、冷却、复仇、每日次数、积分和赛季规则。

PVP 模块可以读取和使用核心资产，但不能绕过核心直接修改玩家长期资产。

长期资产包括：

```text
资源
建筑
兵力
武将
道具
货币
Buff/加成
```

这些资产变化必须通过核心已有入口或新增标准事务入口处理。

## <span style="color:#dc2626">2. 核心原则</span>

PVP 开发必须遵循 Hero3 当前架构规则：

- PVP 是玩法模块，通过标准接口接入核心。
- 战斗计算复用 `internal/core/combat`。
- 攻击方出兵必须从长期兵力中扣出或锁定。
- 防守方自有兵力来自长期兵力。
- 防守方援军来自 `player_reinforcements`，不能写入防守方长期兵力。
- 武将参战必须遵循武将占用规则。
- 战斗损耗必须同步回各自资产归属方。
- 奖励必须走 Reward 或 Effect Pipeline。
- 建筑损坏、保护、占领等长期状态变化必须走建筑变更入口。
- 战报、信函、事件必须作为结算后的标准副作用处理。
- 跨玩家事务必须固定锁顺序，避免死锁。
- 行军到达、战斗结算、奖励发放必须幂等。

不允许：

- PVP 模块直接改玩家兵力表。
- PVP 模块直接把援军当成防守方兵力。
- PVP 模块自己绕过奖励管线发资源、道具、兵力、金币。
- PVP 结算失败后留下“已扣兵但无行军/无战报/无状态”的半完成结果。
- 同一场行军被多个请求重复结算。

## <span style="color:#dc2626">3. 最终功能范围</span>

PVP 最终版包含：

- 玩家目标列表。
- 玩家详情侦查入口预留。
- 发起攻击。
- 行军。
- 行军加速。
- 行军召回。
- 到达结算。
- 防守方自有兵力参战。
- 防守方驻扎援军参战。
- 防守武将参战。
- 援军武将参战。
- 双方战损。
- 援军战损。
- 资源掠夺。
- 建筑损坏预留。
- 战报。
- 信函通知预留。
- 保护期。
- 冷却。
- 免战。
- 复仇。
- 每日攻击次数。
- PVP 积分。
- 赛季排行榜。
- GM 查询和修复能力。

可以后续分阶段落地，但最终数据结构和模块边界应支持这些能力。

## <span style="color:#dc2626">4. 和现有系统的关系</span>

### <span style="color:#dc2626">4.1 战斗系统</span>

PVP 使用核心战斗引擎：

```text
internal/core/combat
```

PVP 只负责组装攻击方、防守方、援军和武将 Buff，不重写战斗公式。

### <span style="color:#dc2626">4.2 增援系统</span>

PVP 防守必须接入增援系统：

```text
BuildDefenseReinforcementUnits(defender_player_id)
ApplyReinforcementBattleResult(report_id, losses)
```

规则：

- 只读取 `stationed` 且 `CanFight = true` 的援军。
- 多批援军按批次进入战斗。
- 每批援军保留来源玩家和增援记录。
- 援军损耗扣对应派出方，不扣防守方。
- 援军全灭后由增援系统负责返程归档和武将释放。

### <span style="color:#dc2626">4.3 战报系统</span>

PVP 必须生成战报。

战报系统后续需要单独升级，但 PVP 从第一版就要提供战报所需字段：

- 攻击方。
- 防守方。
- 攻击方出兵。
- 防守方自有兵力。
- 参战援军。
- 双方武将。
- 援军武将。
- 战斗结果。
- 双方损耗。
- 援军损耗。
- 掠夺资源。
- 积分变化。
- 复仇关系。

### <span style="color:#dc2626">4.4 信函系统</span>

PVP 可以通过信函发送通知，但不应依赖信函完成核心结算。

适合信函通知的场景：

- 被攻击通知。
- 防守胜利通知。
- 防守失败通知。
- 援军参战通知。
- 赛季结算奖励。

奖励仍走信函附件或 Reward/Effect Pipeline，不能由 PVP 直接发放。

## <span style="color:#dc2626">5. PVP 模块边界回答</span>

### <span style="color:#dc2626">5.1 读取什么玩家状态</span>

攻击方：

- 玩家基础信息。
- 资源。
- 兵力。
- 武将。
- 武将占用。
- Buff。
- 保护/冷却/次数状态。

防守方：

- 玩家基础信息。
- 资源。
- 建筑。
- 兵力。
- 武将。
- Buff。
- 保护状态。
- 驻扎援军。

### <span style="color:#dc2626">5.2 修改什么玩家状态</span>

攻击方：

- 扣出出征兵力。
- 行军结束后记录损耗。
- 返还召回剩余兵力。
- 增加掠夺资源或奖励。
- 更新 PVP 次数、冷却、积分。

防守方：

- 扣除防守损耗。
- 扣除被掠夺资源。
- 更新保护期。
- 更新 PVP 积分。

援军派出方：

- 更新援军损耗。
- 释放或返还援军武将。
- 记录援军参战结果。

### <span style="color:#dc2626">5.3 使用或锁定什么资产</span>

- 攻击方出征兵力。
- 攻击方出征武将。
- 防守方自有防守兵力。
- 防守方防守武将。
- 防守方驻扎援军。
- 援军携带武将。
- 双方 PVP 状态记录。

### <span style="color:#dc2626">5.4 发放什么奖励</span>

最终可支持：

- 攻击胜利奖励。
- 防守胜利奖励。
- 资源掠夺。
- 积分奖励。
- 赛季奖励。
- 复仇奖励。

奖励发放原则：

- 即时资源掠夺通过 PVP 结算事务处理。
- 道具、兵力、Buff、金币等奖励走 Reward/Effect Pipeline。
- 赛季奖励优先走信函附件。

### <span style="color:#dc2626">5.5 触发什么事件</span>

建议事件：

```text
pvp.attack_started
pvp.march_recalled
pvp.march_arrived
pvp.battle_resolved
pvp.report_created
pvp.protection_started
pvp.revenge_created
pvp.season_points_changed
pvp.season_settled
```

### <span style="color:#dc2626">5.6 是否需要模块状态</span>

需要。

PVP 必须有自己的模块状态表：

- 行军。
- 战斗。
- 玩家 PVP 状态。
- 复仇。
- 赛季。
- 赛季玩家记录。

## <span style="color:#dc2626">6. 玩法规则</span>

### <span style="color:#dc2626">6.1 攻击目标</span>

玩家可以攻击其他玩家城池。

目标限制：

- 不能攻击自己。
- 不能攻击同账号下其他存档。
- 不能攻击 NPC。NPC 战斗走现有 NPC 系统。
- 不能攻击处于免战保护中的玩家。
- 不能攻击不满足等级或新手保护规则的玩家。

后续可扩展：

- 排行榜目标。
- 附近目标。
- 仇人目标。
- 复仇目标。
- 推荐目标。

### <span style="color:#dc2626">6.2 行军</span>

PVP 使用行军模式，不做纯即时战斗。

行军规则：

- 发起攻击后创建行军记录。
- 出征兵力从攻击方可用兵力中扣出。
- 出征武将进入 PVP 占用状态。
- 行军到达后触发战斗结算。
- 行军中可加速。
- 行军中可召回。
- 到达后不可召回。

行军时间：

```text
基础行军时间按距离或固定规则计算
实际行军时间 = 基础时间 / 行军速度倍率
```

如果当前没有世界地图距离，第一版可使用固定基础时间，保留距离字段。

### <span style="color:#dc2626">6.3 加速</span>

加速规则：

- 只能加速攻击方自己的行军。
- 只能加速 `marching` 状态。
- 加速可以消耗道具或城金。
- 加速不能让剩余时间低于配置下限。
- 每次加速记录次数。

加速消耗：

- 城金消耗走账号/玩家货币标准入口。
- 道具消耗走背包标准入口。

### <span style="color:#dc2626">6.4 召回</span>

召回规则：

- 只能召回自己的行军。
- 只能召回 `marching` 状态。
- 召回后进入 `returning`。
- 返程时间默认等于剩余去程时间，或按配置重新计算。
- 返程完成后返还剩余出征兵力并释放武将。

### <span style="color:#dc2626">6.5 防守组装</span>

防守方战斗力量：

```text
防守方自有兵力
防守方防守武将
防守方 Buff
驻扎援军兵力
援军携带武将
援军自身 Buff
```

援军 Buff 规则：

- 援军武将 Buff 只作用于自己所在援军批次。
- 援军 Buff 不作用于防守方自有兵力。
- 援军 Buff 不作用于其他援军。

### <span style="color:#dc2626">6.6 战斗损耗</span>

损耗归属：

- 攻击方损耗扣攻击方。
- 防守方自有兵力损耗扣防守方。
- 援军损耗扣援军派出方。
- 出征武将默认不死亡。
- 援军武将默认不死亡。

后续可扩展：

- 伤兵。
- 武将受伤。
- 武将被俘。
- 死亡率。
- 医馆治疗。

当前不强制做伤兵系统，但数据结构要预留 `wounded_json`。

### <span style="color:#dc2626">6.7 资源掠夺</span>

攻击胜利可掠夺防守方资源。

掠夺规则：

- 只掠夺可掠夺资源。
- 保底资源不被掠夺。
- 掠夺量受攻击方剩余兵力负重限制。
- 掠夺量受防守方仓库保护影响。
- 掠夺结果写入战报。

推荐公式：

```text
可掠夺量 = max(0, 防守方资源 - 保护量)
实际掠夺量 = min(可掠夺量, 攻击方剩余负重)
```

### <span style="color:#dc2626">6.8 建筑损坏</span>

建筑损坏作为最终能力预留。

规则：

- 只有特定 PVP 模式可造成建筑损坏。
- 建筑损坏必须走建筑生命周期入口。
- 不能由 PVP 直接更新 `player_buildings`。

第一版可不开发建筑损坏，但 `pvp_battles` 预留 `building_effects_json`。

### <span style="color:#dc2626">6.9 保护期和免战</span>

保护类型：

```text
newbie      新手保护
defeat      战败保护
manual      主动免战
system      系统保护
maintenance 维护保护
```

规则：

- 保护期间不能被普通 PVP 攻击。
- 玩家主动攻击别人会打破 `manual`、`newbie`、`defeat` 普通保护，不打破 `system` 和 `maintenance` 保护。
- 战败保护由防守失败触发。
- 免战道具通过 `pvp_protection` 道具效果写入 PVP 状态；第一版已内置 `pvp_truce_8h` 免战令。

### <span style="color:#dc2626">6.10 冷却和每日次数</span>

最终规则：

- 玩家发起攻击后进入攻击冷却。
- 同一目标可有目标冷却。
- 每日攻击次数有限。
- 复仇攻击可以走独立次数或免次数。

建议字段：

```text
daily_attack_count
daily_attack_limit
last_attack_at
cooldown_until
target_cooldown_json
```

### <span style="color:#dc2626">6.11 复仇</span>

防守方被攻击后生成复仇记录。

规则：

- 复仇目标是攻击方。
- 复仇记录有有效期。
- 复仇可突破部分目标限制，但不能突破系统维护保护。
- 复仇成功后记录关闭。
- 同一攻击可只生成一条复仇记录。

### <span style="color:#dc2626">6.12 积分和赛季</span>

最终 PVP 支持赛季。

赛季规则：

- 每个赛季有开始和结束时间。
- 玩家有赛季积分。
- 攻击胜利、防守胜利、失败都会影响积分。
- 赛季结束后按排名发奖励。
- 赛季奖励通过信函发放。

第一版可先只预留表，不做完整赛季结算。

## <span style="color:#dc2626">7. 状态模型</span>

### <span style="color:#dc2626">7.1 行军状态</span>

```text
marching    行军中
returning   召回返程中
resolving   已到达，结算抢占中
resolved    已结算
recalled    已召回完成
cancelled   已取消
failed      异常
```

状态流转：

```text
marching -> resolving -> resolved
marching -> returning -> recalled
marching -> failed
resolving -> failed
```

### <span style="color:#dc2626">7.2 战斗状态</span>

```text
created
resolving
resolved
failed
```

### <span style="color:#dc2626">7.3 玩家 PVP 状态</span>

```text
normal
protected
cooldown
restricted
```

## <span style="color:#dc2626">8. 数据库设计</span>

### <span style="color:#dc2626">8.1 PVP 行军表</span>

表名：`pvp_marches`

```text
id                       VARCHAR(64) PRIMARY KEY
attacker_player_id        VARCHAR(64) NOT NULL
attacker_name             VARCHAR(64) NOT NULL DEFAULT ''
attacker_faction          VARCHAR(32) NOT NULL DEFAULT ''
defender_player_id        VARCHAR(64) NOT NULL
defender_name             VARCHAR(64) NOT NULL DEFAULT ''
defender_faction          VARCHAR(32) NOT NULL DEFAULT ''
march_type                VARCHAR(32) NOT NULL
status                    VARCHAR(32) NOT NULL
attack_troops_json        JSON NOT NULL
attack_generals_json      JSON NULL
speed_multiplier          DECIMAL(10,4) NOT NULL DEFAULT 1
duration_seconds          INT NOT NULL
started_at                DATETIME(6) NOT NULL
arrives_at                DATETIME(6) NOT NULL
return_started_at         DATETIME(6) NULL
returns_at                DATETIME(6) NULL
resolved_at               DATETIME(6) NULL
attacker_report_id        VARCHAR(64) NOT NULL DEFAULT ''
defender_report_id        VARCHAR(64) NOT NULL DEFAULT ''
accelerated_times         INT NOT NULL DEFAULT 0
metadata_json             JSON NULL
created_at                DATETIME(6) NOT NULL
updated_at                DATETIME(6) NOT NULL
```

索引：

```text
idx_pvp_marches_attacker(attacker_player_id, status, arrives_at)
idx_pvp_marches_defender(defender_player_id, status, arrives_at)
idx_pvp_marches_due(status, arrives_at)
```

### <span style="color:#dc2626">8.2 PVP 战斗表</span>

表名：`pvp_battles`

```text
id                         VARCHAR(64) PRIMARY KEY
march_id                   VARCHAR(64) NOT NULL
attacker_player_id          VARCHAR(64) NOT NULL
defender_player_id          VARCHAR(64) NOT NULL
status                      VARCHAR(32) NOT NULL
attacker_snapshot_json      JSON NOT NULL
defender_snapshot_json      JSON NOT NULL
reinforcement_snapshot_json JSON NULL
result_json                 JSON NULL
losses_json                 JSON NULL
plunder_json                JSON NULL
building_effects_json       JSON NULL
points_delta_json           JSON NULL
attacker_report_id          VARCHAR(64) NOT NULL DEFAULT ''
defender_report_id          VARCHAR(64) NOT NULL DEFAULT ''
resolved_at                 DATETIME(6) NULL
created_at                  DATETIME(6) NOT NULL
updated_at                  DATETIME(6) NOT NULL
```

索引：

```text
idx_pvp_battles_march(march_id)
idx_pvp_battles_attacker(attacker_player_id, created_at)
idx_pvp_battles_defender(defender_player_id, created_at)
idx_pvp_battles_status(status, created_at)
```

### <span style="color:#dc2626">8.3 玩家 PVP 状态表</span>

表名：`pvp_player_states`

```text
player_id              VARCHAR(64) PRIMARY KEY
status                 VARCHAR(32) NOT NULL DEFAULT 'normal'
protection_type         VARCHAR(32) NOT NULL DEFAULT ''
protected_until         DATETIME(6) NULL
cooldown_until          DATETIME(6) NULL
daily_attack_count      INT NOT NULL DEFAULT 0
daily_attack_limit      INT NOT NULL DEFAULT 0
daily_reset_at          DATETIME(6) NULL
target_cooldown_json    JSON NULL
metadata_json           JSON NULL
created_at              DATETIME(6) NOT NULL
updated_at              DATETIME(6) NOT NULL
```

### <span style="color:#dc2626">8.4 复仇表</span>

表名：`pvp_revenge_records`

```text
id                  VARCHAR(64) PRIMARY KEY
source_battle_id    VARCHAR(64) NOT NULL
defender_player_id  VARCHAR(64) NOT NULL
attacker_player_id  VARCHAR(64) NOT NULL
status              VARCHAR(32) NOT NULL
expires_at          DATETIME(6) NOT NULL
used_at             DATETIME(6) NULL
created_at          DATETIME(6) NOT NULL
updated_at          DATETIME(6) NOT NULL
```

索引：

```text
idx_pvp_revenge_defender(defender_player_id, status, expires_at)
idx_pvp_revenge_attacker(attacker_player_id, status, expires_at)
```

### <span style="color:#dc2626">8.5 赛季表</span>

表名：`pvp_seasons`

```text
id             VARCHAR(64) PRIMARY KEY
name           VARCHAR(120) NOT NULL
status         VARCHAR(32) NOT NULL
starts_at      DATETIME(6) NOT NULL
ends_at        DATETIME(6) NOT NULL
settled_at     DATETIME(6) NULL
rules_json     JSON NULL
rewards_json   JSON NULL
created_at     DATETIME(6) NOT NULL
updated_at     DATETIME(6) NOT NULL
```

表名：`pvp_season_players`

```text
season_id       VARCHAR(64) NOT NULL
player_id       VARCHAR(64) NOT NULL
points          BIGINT NOT NULL DEFAULT 0
rating          BIGINT NOT NULL DEFAULT 0
wins            INT NOT NULL DEFAULT 0
losses          INT NOT NULL DEFAULT 0
defense_wins    INT NOT NULL DEFAULT 0
defense_losses  INT NOT NULL DEFAULT 0
last_battle_at  DATETIME(6) NULL
reward_sent_at  DATETIME(6) NULL
created_at      DATETIME(6) NOT NULL
updated_at      DATETIME(6) NOT NULL
PRIMARY KEY (season_id, player_id)
```

索引：

```text
idx_pvp_season_points(season_id, points)
idx_pvp_season_rating(season_id, rating)
idx_pvp_season_player(player_id, updated_at)
```

## <span style="color:#dc2626">9. 后端代码组织</span>

应用层：

```text
go/internal/app/game/pvp.go
go/internal/app/game/service_pvp.go
go/internal/app/game/pvp_march.go
go/internal/app/game/pvp_defense_builder.go
go/internal/app/game/pvp_loss_settlement.go
go/internal/app/game/pvp_protection.go
go/internal/app/game/pvp_season.go
```

存储层：

```text
go/internal/infrastructure/storage/mysql_pvp.go
```

接口层：

```text
go/internal/transport/api/handlers_pvp.go
go/internal/transport/api/handlers_admin_pvp.go
```

注册：

```text
go/internal/app/game/gameplay_module_registry.go
```

## <span style="color:#dc2626">10. Repository 接口</span>

建议新增：

```text
CreatePvpMarchWithState
ClaimPvpMarchForResolution
UpdatePvpMarch
GetPvpMarch
ListPvpMarchesForPlayer
ListDuePvpMarches
CreatePvpBattle
UpdatePvpBattle
GetPvpBattle
ListPvpBattlesForPlayer
GetPvpPlayerState
UpdatePvpPlayerState
CreatePvpRevengeRecord
ListPvpRevengeRecords
UpdatePvpSeasonPlayer
```

跨玩家结算建议有专用事务入口：

```text
ResolvePvpBattleTransaction(marchID, fn)
```

事务职责：

- 抢占行军。
- 锁攻击方。
- 锁防守方。
- 锁必要援军记录。
- 写 PVP 战斗记录。
- 写双方玩家状态。
- 写行军状态。
- 写战报 ID。

## <span style="color:#dc2626">11. 接口设计</span>

### <span style="color:#dc2626">11.1 玩家接口</span>

目标列表：

```text
GET /api/v1/pvp/targets
```

目标详情：

```text
GET /api/v1/pvp/targets/{playerId}
```

发起攻击：

```text
POST /api/v1/pvp/attacks
```

请求：

```text
targetPlayerId
troops
generalIds
marchMode
```

我的行军：

```text
GET /api/v1/pvp/marches
```

行军加速：

```text
POST /api/v1/pvp/marches/{marchId}/accelerate
```

行军召回：

```text
POST /api/v1/pvp/marches/{marchId}/recall
```

战斗列表：

```text
GET /api/v1/pvp/battles
```

战斗详情：

```text
GET /api/v1/pvp/battles/{battleId}
```

PVP 状态：

```text
GET /api/v1/pvp/state
```

复仇列表：

```text
GET /api/v1/pvp/revenge
```

赛季信息：

```text
GET /api/v1/pvp/season
GET /api/v1/pvp/rankings
```

### <span style="color:#dc2626">11.2 GM 接口</span>

```text
GET /api/v1/admin/pvp/marches
GET /api/v1/admin/pvp/battles
GET /api/v1/admin/pvp/players/{playerId}
GET /api/v1/admin/pvp/seasons
POST /api/v1/admin/pvp/seasons
PUT /api/v1/admin/pvp/seasons/{seasonId}
POST /api/v1/admin/pvp/marches/{marchId}/force-resolve
POST /api/v1/admin/pvp/marches/{marchId}/cancel
```

GM 主要用于查询、排查和修复，不应成为玩家资产随意修改入口。

当前第一版修复动作限制：

- 强制结算只处理 `marching` 或 `resolving` 行军。
- 取消行军只处理尚未生成战斗的 `marching`、`returning` 或 `resolving` 行军。
- 取消行军会返还攻击方出征兵力并释放 PVP 出征武将。
- 已 `resolved`、`recalled`、`cancelled` 或已经生成 `battleId` 的行军不能再次取消。

## <span style="color:#dc2626">12. 核心流程</span>

### <span style="color:#dc2626">12.1 发起攻击</span>

```text
校验攻击方
校验防守方
校验保护期
校验冷却
校验每日次数
校验同账号限制
校验兵力和武将
锁攻击方资产
扣出攻击兵力
占用攻击武将
创建 pvp_marches
更新 PVP 次数和冷却
返回行军
```

### <span style="color:#dc2626">12.2 到达结算</span>

```text
查询到期行军
抢占行军状态 marching -> resolving
锁攻击方、防守方、相关援军
结算防守方资源、兵力、Buff
构建攻击方快照
构建防守方自有快照
构建援军快照
调用战斗引擎
计算损耗
计算掠夺
计算积分
写攻击方损耗和奖励
写防守方损耗和资源变化
写援军损耗
生成战报
写 pvp_battles
标记行军 resolved
释放攻击武将
发布事件
```

### <span style="color:#dc2626">12.3 召回完成</span>

```text
行军 returning 到期
锁攻击方
返还剩余出征兵力
释放武将
标记 recalled
```

## <span style="color:#dc2626">13. 跨玩家事务和锁顺序</span>

PVP 是高风险跨玩家业务。

必须固定锁顺序：

```text
按 player_id 字典序锁所有相关玩家
```

相关玩家包括：

- 攻击方。
- 防守方。
- 所有参战援军派出方。

建议：

- 行军抢占先独立完成。
- 战斗计算尽量基于快照。
- 锁内只做最终校验和落库。
- 所有结算步骤必须可重试。

## <span style="color:#dc2626">14. 幂等设计</span>

必须幂等：

- 创建行军。
- 行军加速。
- 行军召回。
- 行军到达抢占。
- 战斗结算。
- 援军损耗写入。
- 战报写入。
- 积分变更。
- 复仇记录生成。
- 赛季奖励发放。

推荐做法：

- 行军状态使用条件更新抢占。
- 战斗表 `march_id` 唯一。
- 战报 ID 写入行军表。
- 援军损耗按 `battle_id + reinforcement_id + unit_type` 生成可追踪明细。
- 赛季奖励通过信函发放，并记录 `reward_sent_at`。

## <span style="color:#dc2626">15. 前端页面设计</span>

玩家端入口：

```text
左侧菜单：PVP
```

页面：

- 目标列表。
- 目标详情。
- 发起攻击弹窗。
- 我的行军。
- 战斗记录。
- 复仇列表。
- 赛季排行。
- 我的 PVP 状态。

目标卡片：

- 玩家名。
- 阵营。
- 战力或兵力摘要。
- 是否可攻击。
- 保护状态。
- 复仇标识。
- 攻击按钮。

行军卡片：

- 目标玩家。
- 状态。
- 到达倒计时。
- 出征兵力。
- 出征武将。
- 加速按钮。
- 召回按钮。

战斗记录：

- 胜负。
- 攻击方。
- 防守方。
- 损耗摘要。
- 掠夺摘要。
- 战报入口。

## <span style="color:#dc2626">16. GM 后台页面设计</span>

GM 后台建议：

- PVP 行军查询。当前已完成只读接口和页面。
- PVP 战斗查询。当前已完成只读接口和页面。
- 玩家 PVP 状态查询。当前已完成只读接口和页面。
- 赛季管理。当前已完成赛季查询、创建、编辑和结算接口；后台页面已提供查询和结算入口。
- 异常行军修复。当前已支持强制结算和安全取消未结算行军。

不建议第一时间做：

- GM 任意改积分。
- GM 任意改胜负。
- GM 手动制造战斗。

这些能力风险较高，应等审计系统完善后再考虑。

## <span style="color:#dc2626">17. 测试验收</span>

必须覆盖：

- 不能攻击自己。
- 不能攻击同账号存档。
- 不能攻击保护中玩家。
- 发起攻击扣出攻击方兵力。
- 发起攻击占用武将。
- 行军到达可抢占。
- 重复抢占只成功一次。
- 防守方自有兵力参与战斗。
- 防守方援军参与战斗。
- 援军损耗归属派出方。
- 攻击方损耗正确。
- 防守方损耗正确。
- 掠夺资源不超过保护量和负重。
- 战报生成。
- 战斗表和行军表状态一致。
- 召回返还兵力且释放武将。
- 加速消耗正确。
- 冷却和每日次数生效。
- 复仇记录生成。
- 积分变化正确。
- 赛季排行可查询。

并发测试：

- A 攻击 B，B 同时攻击 A。
- 多人同时攻击同一玩家。
- 多批援军同时参与防守。
- 同一行军被多个请求同时结算。
- 战报保存失败时不重复扣兵。

## <span style="color:#dc2626">18. 运维和修复工具</span>

建议 dbtool 增加：

```text
verify-pvp-marches
verify-pvp-battles
repair-pvp-march
repair-pvp-battle
backfill-pvp-player-states
settle-due-pvp-marches
```

校验内容：

- `marching` 到期但未结算。
- `resolving` 卡住超过阈值。
- `resolved` 但没有 battle。
- battle 有结果但行军未写 report ID。
- 兵力扣出后行军缺失。
- 战报缺失。

线上修复必须遵循迁移协作规则，不能未经确认直接改线上库。

## <span style="color:#dc2626">19. 分阶段落地建议</span>

虽然本文档是最终版设计，但开发可以按下面顺序落地：

1. PVP 模块登记、表结构、Repository。
2. 目标列表和发起攻击。
3. 行军、召回、加速。
4. 到达抢占和基础战斗结算。
5. 接入增援防守。
6. 战报升级。
7. 掠夺资源。
8. 保护期、冷却、每日次数。
9. 复仇。
10. 积分和排行榜。
11. 赛季和赛季奖励信函。
12. GM 排查和修复工具。

## <span style="color:#dc2626">20. 最终交付标准</span>

PVP 系统完成时，应达到：

```text
玩家可以选择目标并发起 PVP 攻击
攻击以行军形式存在
攻击方兵力和武将正确扣出/占用/返还/释放
行军可以加速和召回
行军到达后幂等结算
防守方自有兵力和驻扎援军共同参战
援军损耗按来源批次归属
战斗结果生成完整战报
攻击方、防守方、援军派出方资产变化正确
资源掠夺遵循保护量和负重
保护期、免战、冷却、每日次数、复仇生效
积分和赛季排行可用
赛季奖励通过信函发放
GM 可以查询和排查 PVP 行军、战斗和玩家 PVP 状态
所有跨玩家结算有固定锁顺序和幂等保护
PVP 模块不绕过核心长期资产入口
```
