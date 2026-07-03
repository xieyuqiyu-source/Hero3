# <span style="color:#16a34a">Hero3 增援与 PVP 系统执行文档</span>

最后更新：2026-06-26

## <span style="color:#16a34a">1. 文档目标</span>

本文档用于指导 Hero3 增援系统和 PVP 系统的分阶段开发。

本次设计遵循项目当前核心方向：

- 后端保持模块化单体。
- 核心长期资产保持稳定。
- 玩法模块通过标准接口接入核心。
- 增援、PVP 不直接绕过核心修改玩家兵力、武将、资源、建筑、道具、货币等长期资产。
- 所有长期资产变化优先通过 Effect Pipeline、奖励管线、建筑变更、战斗结算和标准事务入口执行。

## <span style="color:#16a34a">2. 总体结论</span>

增援系统和 PVP 系统应拆成两个玩法模块，但共享一套军事行动状态。

推荐模块划分：

```text
military_dispatch  通用军事行动状态，负责出征、行军、驻扎、返回和行动生命周期
reinforcement      增援玩法模块，负责派遣援军、驻防、召回、参与防守和损耗归属
pvp                PVP 玩法模块，负责挑战、匹配、保护、积分、结算和战斗记录
```

第一阶段不要直接做完整世界地图、联盟战、赛季排行榜和复杂行军策略。先把下面主链路打通：

```text
玩家派出兵力 -> 兵力离开可用池 -> 到达目标 -> 参与防守或攻击 -> 战斗损耗 -> 剩余兵力返回或继续驻防
```

这条链路是后续 PVP、攻城、驻防、国战、远征、采集和侦查的共同基础。

## <span style="color:#16a34a">3. 核心边界</span>

### <span style="color:#16a34a">3.1 哪些属于核心</span>

核心或应用层基础系统应负责：

- 玩家兵力长期资产。
- 玩家武将长期资产。
- 资源、建筑、道具、货币等长期资产。
- 标准战斗引擎。
- 标准奖励发放。
- 标准建筑状态变更。
- 标准事件管线。
- 标准战报和通知接口。
- 玩家状态事务读写。

### <span style="color:#16a34a">3.2 哪些属于玩法模块</span>

增援模块负责：

- 援军派遣条件。
- 援军行军和驻扎状态。
- 援军召回规则。
- 援军参与防守规则。
- 援军损耗归属。
- 援军返回规则。

PVP 模块负责：

- 攻击目标选择。
- 攻击条件和冷却。
- 防守快照组装。
- 保护期和限制规则。
- PVP 积分和记录。
- 战斗结果解释。
- PVP 奖励和惩罚策略。

### <span style="color:#16a34a">3.3 不允许的实现</span>

下面做法不允许作为正式实现：

- 增援到达后直接把派出方兵力写入目标玩家兵力表。
- PVP 模块直接修改玩家资源、兵力、建筑状态。
- PVP 模块自己拼接奖励结果并绕过奖励管线。
- 增援损耗只写在增援表中，不同步回派出方真实兵力。
- 出征状态只存在于前端或临时内存。
- 战斗结算失败时已经扣兵但没有可恢复记录。

如果现有核心无法支持“兵力锁定、派出、返回、损耗同步”，应先补核心或军事应用层基础能力，再继续实现增援和 PVP。

## <span style="color:#16a34a">4. 通用军事行动模型</span>

### <span style="color:#16a34a">4.1 设计目的</span>

通用军事行动用于表示一支部队从玩家长期资产中临时离开，进入某个玩法模块控制的行动状态。

它不直接代表新玩法，而是给增援、PVP、远征、侦查、攻城等玩法复用。

### <span style="color:#16a34a">4.2 行动状态</span>

建议使用以下标准状态：

```text
preparing   已创建，尚未出发
marching    行军中
stationed   已驻扎
fighting    战斗处理中
returning   返回中
completed   已完成
cancelled   已取消
failed      异常终止，需要运维或修复流程处理
```

### <span style="color:#16a34a">4.3 建议数据表</span>

表名：`player_military_dispatches`

```text
id                    BIGINT PK
dispatch_id           VARCHAR(64) UNIQUE
module_id             VARCHAR(64)
owner_player_id        VARCHAR(64)
target_player_id       VARCHAR(64) NULL
target_type            VARCHAR(64)
target_id              VARCHAR(64)
status                 VARCHAR(32)
troops_json            JSON
generals_json          JSON
started_at             DATETIME
arrive_at              DATETIME NULL
return_at              DATETIME NULL
completed_at           DATETIME NULL
battle_report_id       VARCHAR(64) NULL
metadata_json          JSON
created_at             DATETIME
updated_at             DATETIME
```

核心索引：

```text
UNIQUE KEY uk_dispatch_id(dispatch_id)
KEY idx_owner_status(owner_player_id, status)
KEY idx_target_status(target_type, target_id, status)
KEY idx_module_status(module_id, status)
KEY idx_arrive_at(status, arrive_at)
KEY idx_return_at(status, return_at)
```

### <span style="color:#16a34a">4.4 兵力快照原则</span>

`troops_json` 保存行动创建时派出的兵力快照。

注意：

- 快照用于行动结算和战报展示。
- 真实长期兵力仍归核心资产系统管理。
- 战斗损耗必须通过标准入口同步回派出方。
- 返回兵力必须通过标准入口回到派出方可用兵力。

### <span style="color:#16a34a">4.5 军事行动服务入口</span>

建议在应用层建立通用服务能力：

```text
CreateDispatch        创建军事行动并锁定或扣出派出兵力
StartDispatch         行动出发
MarkArrived           标记到达
MarkFighting          标记进入战斗
ApplyDispatchLosses   应用行动损耗
ReturnDispatch        开始返回
CompleteDispatch      完成行动并返还剩余兵力
CancelDispatch        取消未出发行动
ListDispatches        查询玩家行动
```

服务内部必须使用玩家状态事务，避免并发扣兵和重复返还。

## <span style="color:#16a34a">5. 增援系统设计</span>

### <span style="color:#16a34a">5.1 模块定位</span>

增援系统是一个玩法模块，用于让玩家把自己的部队派往另一个玩家或目标，作为外援防守力量参与战斗。

增援兵力的归属始终属于派出方，不属于接收方。

### <span style="color:#16a34a">5.2 业务规则 MVP</span>

第一版只实现玩家之间增援：

- 玩家可以选择目标玩家进行增援。
- 玩家可以选择派出的兵种数量。
- 第一版可以暂不派武将，或只允许派一个武将。
- 派出后兵力从派出方可用兵力中离开。
- 到达后援军进入目标玩家驻防列表。
- 目标玩家被 PVP 攻击时，有效援军参与防守。
- 战斗损耗按援军来源拆回派出方。
- 派出方可以召回未战斗或战后剩余援军。
- 接收方可以查看援军，但不能把援军作为自己的长期兵力使用。

### <span style="color:#16a34a">5.3 建议数据表</span>

表名：`player_reinforcements`

```text
id                    BIGINT PK
reinforcement_id      VARCHAR(64) UNIQUE
dispatch_id           VARCHAR(64)
from_player_id         VARCHAR(64)
to_player_id           VARCHAR(64)
status                 VARCHAR(32)
troops_json            JSON
generals_json          JSON
losses_json            JSON
sent_at                DATETIME
arrived_at             DATETIME NULL
recalled_at            DATETIME NULL
returned_at            DATETIME NULL
last_battle_report_id  VARCHAR(64) NULL
metadata_json          JSON
created_at             DATETIME
updated_at             DATETIME
```

推荐状态：

```text
marching
stationed
fighting
returning
returned
cancelled
failed
```

核心索引：

```text
UNIQUE KEY uk_reinforcement_id(reinforcement_id)
UNIQUE KEY uk_dispatch_id(dispatch_id)
KEY idx_from_status(from_player_id, status)
KEY idx_to_status(to_player_id, status)
KEY idx_arrived_status(to_player_id, status, arrived_at)
```

### <span style="color:#16a34a">5.4 增援接口 MVP</span>

玩家侧接口建议：

```text
POST /api/v1/reinforcements
GET  /api/v1/reinforcements/sent
GET  /api/v1/reinforcements/received
POST /api/v1/reinforcements/{id}/recall
```

GM 侧接口建议：

```text
GET  /api/v1/admin/reinforcements
GET  /api/v1/admin/reinforcements/{id}
POST /api/v1/admin/reinforcements/{id}/force-return
```

### <span style="color:#16a34a">5.5 增援事件</span>

建议注册标准事件：

```text
reinforcement.sent
reinforcement.arrived
reinforcement.recalled
reinforcement.used_in_battle
reinforcement.returned
reinforcement.failed
```

事件载荷至少包含：

```text
reinforcement_id
dispatch_id
from_player_id
to_player_id
status
troops_summary
occurred_at
```

### <span style="color:#16a34a">5.6 增援验收标准</span>

第一版完成时必须满足：

- 派出增援后，派出方可用兵力减少或被锁定。
- 同一批兵不能被重复派出。
- 援军到达后，接收方防守列表可查。
- 接收方不能直接消耗援军。
- 召回后，剩余兵力回到派出方。
- 战斗损耗后，派出方真实兵力正确减少。
- 重复召回、重复到达、重复返还不会复制兵。
- 所有状态变化有测试覆盖。

## <span style="color:#16a34a">6. PVP 系统设计</span>

### <span style="color:#16a34a">6.1 模块定位</span>

PVP 系统是玩家之间战斗玩法模块。

PVP 模块负责战斗发起条件、目标选择、保护规则、积分规则、战斗记录和奖励策略。实际战斗计算复用核心战斗引擎，长期资产变化通过标准核心入口执行。

### <span style="color:#16a34a">6.2 PVP MVP 范围</span>

第一版建议只做即时挑战结算：

- 玩家选择一个目标玩家。
- 攻击方选择出战兵力。
- 后端校验攻击条件。
- 后端创建 PVP 战斗记录。
- 后端扣出或锁定攻击方兵力。
- 后端读取防守方当前防守力量。
- 后端合并防守方有效增援。
- 调用战斗引擎结算。
- 写入战报。
- 同步双方兵力损耗。
- 返回战斗结果。

暂不做：

- 赛季排行榜。
- 自动匹配。
- 复杂保护罩。
- 世界地图行军。
- 侦查。
- 资源大规模掠夺。
- 联盟宣战。

### <span style="color:#16a34a">6.3 建议数据表</span>

表名：`pvp_battles`

```text
id                    BIGINT PK
pvp_battle_id          VARCHAR(64) UNIQUE
dispatch_id            VARCHAR(64) NULL
attacker_id            VARCHAR(64)
defender_id            VARCHAR(64)
status                 VARCHAR(32)
attack_troops_json     JSON
attack_generals_json   JSON
defense_snapshot_json  JSON
reinforcement_json     JSON
result_json            JSON
battle_report_id       VARCHAR(64) NULL
started_at             DATETIME
resolved_at            DATETIME NULL
metadata_json          JSON
created_at             DATETIME
updated_at             DATETIME
```

推荐状态：

```text
created
fighting
resolved
cancelled
failed
```

核心索引：

```text
UNIQUE KEY uk_pvp_battle_id(pvp_battle_id)
KEY idx_attacker_started(attacker_id, started_at)
KEY idx_defender_started(defender_id, started_at)
KEY idx_status_started(status, started_at)
KEY idx_battle_report_id(battle_report_id)
```

### <span style="color:#16a34a">6.4 后续赛季表</span>

赛季和排行榜第二阶段再做。

表名：`pvp_player_seasons`

```text
id              BIGINT PK
season_id       VARCHAR(64)
player_id       VARCHAR(64)
rating          BIGINT
points          BIGINT
wins            BIGINT
losses          BIGINT
draws           BIGINT
last_battle_at  DATETIME NULL
created_at      DATETIME
updated_at      DATETIME
```

核心索引：

```text
UNIQUE KEY uk_season_player(season_id, player_id)
KEY idx_season_points(season_id, points)
KEY idx_season_rating(season_id, rating)
```

### <span style="color:#16a34a">6.5 PVP 接口 MVP</span>

玩家侧接口建议：

```text
GET  /api/v1/pvp/targets
POST /api/v1/pvp/battles
GET  /api/v1/pvp/battles
GET  /api/v1/pvp/battles/{id}
```

请求 `POST /api/v1/pvp/battles` 至少包含：

```text
defender_id
troops
general_ids
```

返回至少包含：

```text
pvp_battle_id
battle_report_id
result
attacker_losses
defender_losses
reinforcement_losses
rewards
state
```

GM 侧接口建议：

```text
GET /api/v1/admin/pvp/battles
GET /api/v1/admin/pvp/battles/{id}
GET /api/v1/admin/pvp/players/{player_id}
```

### <span style="color:#16a34a">6.6 PVP 事件</span>

建议注册标准事件：

```text
pvp.battle_created
pvp.battle_resolved
pvp.report_created
pvp.reward_granted
pvp.failed
```

事件载荷至少包含：

```text
pvp_battle_id
attacker_id
defender_id
battle_report_id
result
occurred_at
```

### <span style="color:#16a34a">6.7 PVP 验收标准</span>

第一版完成时必须满足：

- 攻击方不能派出超过可用兵力的部队。
- 攻击方出战兵力不会被重复使用。
- 防守方兵力和有效增援都能进入战斗。
- 战斗结果可生成战报。
- 双方损耗能正确同步到各自长期兵力。
- 增援损耗能正确同步到增援派出方。
- 战斗失败或事务失败不会出现扣兵后无战报的半完成状态。
- PVP 模块不直接绕过核心改资源、兵力、建筑、奖励。

## <span style="color:#16a34a">7. 防守力量组装</span>

PVP 防守方力量由三部分组成：

```text
防守方自有可防守兵力
防守方可用防守武将
已到达并有效驻扎的增援兵力
```

组装流程：

1. 读取防守方玩家状态。
2. 读取防守方有效增援列表。
3. 过滤状态不是 `stationed` 的增援。
4. 过滤已召回、返回、异常或过期的增援。
5. 将增援按来源保留归属信息。
6. 转换成战斗引擎需要的防守单位。
7. 战斗结束后按单位来源拆分损耗。

防守快照必须写入 `pvp_battles.defense_snapshot_json`，用于战报追溯和异常排查。

## <span style="color:#16a34a">8. 损耗归属规则</span>

损耗结算必须按来源拆分。

推荐来源类型：

```text
self            玩家自己的防守或攻击部队
reinforcement   其他玩家派来的增援部队
npc             后续扩展 NPC 或系统部队
temporary       活动临时部队
```

每个战斗单位进入战斗前应带上来源信息：

```text
source_type
source_player_id
source_record_id
unit_type
amount
```

战斗结束后生成损耗清单：

```text
source_type
source_player_id
source_record_id
unit_type
lost_amount
remaining_amount
```

然后通过标准事务入口同步：

- 攻击方损耗同步到攻击方。
- 防守方自有损耗同步到防守方。
- 增援损耗同步到增援派出方和增援记录。

## <span style="color:#16a34a">9. Effect Pipeline 接入</span>

PVP 和增援模块只能提交标准效果，不直接写长期资产。

第一版可能需要的效果：

```text
reward              奖励发放
building_mutation   建筑损坏、保护、修复等状态变化
modifier            临时 Buff 或 DeBuff
```

如果当前 Effect Pipeline 不支持兵力损耗或兵力返还，应先判断：

- 兵力损耗是否应作为战斗结算专用入口。
- 兵力返还是否应作为军事行动服务入口。
- 是否需要新增标准 Effect 类型，例如 `army_delta`。

不要让 PVP 模块自己直接改兵力表。

## <span style="color:#16a34a">10. 后端代码组织建议</span>

建议新增或整理以下文件。实际命名以项目已有风格为准。

应用层：

```text
go/internal/app/game/gameplay_module_registry.go
go/internal/app/game/military_dispatch.go
go/internal/app/game/military_dispatch_state.go
go/internal/app/game/service_reinforcement.go
go/internal/app/game/service_pvp.go
go/internal/app/game/pvp_defense_builder.go
go/internal/app/game/pvp_loss_settlement.go
```

存储层：

```text
go/internal/infrastructure/storage/mysql_military_dispatch.go
go/internal/infrastructure/storage/mysql_reinforcement.go
go/internal/infrastructure/storage/mysql_pvp.go
```

接口层：

```text
go/internal/transport/api/handlers_reinforcement.go
go/internal/transport/api/handlers_pvp.go
go/internal/transport/api/handlers_admin_reinforcement.go
go/internal/transport/api/handlers_admin_pvp.go
```

前端：

```text
web/src/api/reinforcement.ts
web/src/api/pvp.ts
```

Admin：

```text
admin/src/api/reinforcement.ts
admin/src/api/pvp.ts
```

## <span style="color:#16a34a">11. 开发阶段</span>

### <span style="color:#16a34a">阶段一：通用军事行动地基</span>

目标：

- 新增 `player_military_dispatches`。
- 新增军事行动服务。
- 支持创建、到达、返回、完成、失败状态。
- 接入兵力离开可用池的标准能力。

必须完成：

- 数据库迁移。
- Repository 方法。
- Service 方法。
- 状态机测试。
- 并发扣兵测试。
- 重复完成不复制兵测试。
- 文档更新。

验收：

- 创建行动后，派出兵力不能再次被派出。
- 完成行动后，剩余兵力只返还一次。
- 异常状态可以被查询和排查。

### <span style="color:#16a34a">阶段二：增援 MVP</span>

目标：

- 新增 `player_reinforcements`。
- 玩家可以派出援军。
- 援军可以到达、驻扎、召回、返回。
- 接收方可以查看收到的援军。

必须完成：

- 增援模块登记。
- 增援 Service。
- 增援 Repository。
- 玩家 API。
- GM 查询 API。
- 前端基础入口。
- 测试覆盖派遣、到达、召回、返回。

验收：

- 派出方兵力减少或锁定。
- 接收方不能占有援军。
- 援军返回后派出方兵力正确恢复。
- 重复召回不会重复返还。

### <span style="color:#16a34a">阶段三：PVP 即时结算 MVP</span>

目标：

- 新增 `pvp_battles`。
- 玩家可以挑战另一个玩家。
- PVP 复用战斗引擎。
- 战斗结果生成战报。
- 双方损耗正确同步。

必须完成：

- PVP 模块登记。
- PVP Service。
- PVP Repository。
- 防守力量组装器。
- 损耗结算器。
- 玩家 API。
- GM 查询 API。
- OpenAPI 更新。
- 前端基础入口。

验收：

- 攻击方不能超额出兵。
- 战斗后双方兵力正确变化。
- 战报可查。
- 失败事务不产生半完成状态。

### <span style="color:#16a34a">阶段四：PVP 接入增援</span>

目标：

- PVP 防守方读取有效增援。
- 增援参与防守。
- 增援损耗同步回派出方。
- 增援战斗记录可追溯。

必须完成：

- 防守组装保留来源信息。
- 战斗单位保留来源信息。
- 损耗按来源拆分。
- 更新增援记录的剩余兵力和最近战报。
- 增援派出方收到战报或信函通知。

验收：

- 援军能参与防守。
- 援军损耗不扣接收方兵力。
- 援军损耗正确扣派出方兵力。
- 战报能看到援军来源和损耗。

### <span style="color:#16a34a">阶段五：玩法规则增强</span>

目标：

- 增加攻击冷却。
- 增加简单保护期。
- 增加战斗记录分页。
- 增加 PVP 奖励。
- 增加基础积分。

建议后置内容：

- 赛季排行榜。
- 自动匹配。
- 侦查。
- 世界地图行军。
- 联盟协防限制。
- 攻城和建筑损坏。

## <span style="color:#16a34a">12. 测试清单</span>

后端单元测试：

- 军事行动状态机。
- 兵力派出扣减。
- 兵力返回返还。
- 重复请求幂等。
- 增援派遣。
- 增援召回。
- PVP 攻击参数校验。
- PVP 防守力量组装。
- PVP 损耗归属拆分。
- PVP 战报生成。

后端集成测试：

- 玩家 A 增援玩家 B。
- 玩家 C 攻击玩家 B。
- 玩家 B 自有兵力和玩家 A 援军共同防守。
- 战斗后玩家 A、B、C 兵力都正确。
- 战报和增援记录都可追溯。

数据库校验：

- 行动表没有卡在 `fighting` 的过期记录。
- 增援表没有已返回但行动未完成的记录。
- PVP 战斗没有已结算但无战报的记录。
- 派出兵力、剩余兵力和玩家真实兵力能对账。

前端验证：

- 玩家可以选择兵力派出增援。
- 玩家可以查看派出的援军。
- 玩家可以查看收到的援军。
- 玩家可以召回援军。
- 玩家可以选择目标发起 PVP。
- 玩家可以查看 PVP 结果和战报。

## <span style="color:#16a34a">13. 运维和修复工具</span>

建议在 `dbtool` 中逐步增加：

```text
verify-dispatches
verify-reinforcements
verify-pvp-battles
repair-stuck-dispatch
repair-stuck-reinforcement
repair-pvp-battle
```

工具职责：

- 校验行动状态是否和模块状态一致。
- 校验已完成行动是否重复返还。
- 校验 PVP 战斗是否有战报。
- 校验援军损耗是否同步到派出方。
- 修复异常卡住状态时必须输出明确报告。

线上修复必须遵循项目迁移协作规则，不能未经确认直接改线上库。

## <span style="color:#16a34a">14. 文档同步要求</span>

开发过程中必须同步维护：

- 本文档。
- [数据库设计](../数据库/数据库设计.md)。
- [总体架构设计](../架构/总体架构设计.md)。
- [核心地基设计](../架构/核心地基设计.md)。
- [玩家状态读写模型规范](../架构/玩家状态读写模型规范.md)。
- [OpenAPI 维护规范](../接口文档/OpenAPI维护规范.md)。
- `docs/接口文档/openapi/openapi.yaml`。
- `README.md` 和相关子目录 README。

如果实现中发现现有核心无法表达兵力锁定、派出、损耗和返回，应先暂停玩法推进，补充核心或军事应用层基础能力，再继续实现。

## <span style="color:#16a34a">15. 最小可交付版本</span>

最小可交付版本定义为：

```text
通用军事行动状态
增援派遣、到达、召回、返回
PVP 即时挑战
PVP 防守接入增援
战斗战报
双方和援军损耗正确同步
基础玩家界面
基础 GM 查询界面
核心测试通过
OpenAPI 和中文文档同步
```

达到这个标准后，再进入赛季、排行榜、保护罩、世界地图行军和联盟规则。
