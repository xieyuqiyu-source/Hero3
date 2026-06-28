# Hero3 Go 后端

`Hero3 Go 后端` 是 Hero3 的数据与游戏规则服务。当前采用模块化单体方向：HTTP 输入输出、应用编排、通用核心能力和基础设施分层维护。

当前后端已经接入账号、玩家存档、资源、建筑、征兵、NPC、战斗、武将、道具、信函、公告、万象幻境、货币流水等基础能力，并正在建设稳定核心地基。核心地基要求玩家长期资产变更通过统一玩家状态事务、账号资产事务、奖励发放、事件管线和 Modifier 加成管线处理。

## 技术选择

当前骨架优先使用 Go 标准库，并接入 MySQL/MariaDB 作为可选持久化：

- `net/http`：HTTP 服务与路由
- `log/slog`：结构化日志
- 环境变量：基础配置
- `database/sql` + `github.com/go-sql-driver/mysql`：账号与存档持久化

后续在业务需要明确后，再接入：

- Redis：在线状态、排行榜、短期缓存或队列

## 目录结构

```text
go/
├── cmd/server/             # 服务启动入口
├── internal/transport/api/ # HTTP 路由与接口处理
├── internal/app/game/      # 游戏应用服务与业务编排
├── internal/core/          # 通用领域能力，当前包含战斗、建筑生命周期、Effect、将领事件、Modifier、事件总线、奖励契约和注册中心
├── internal/platform/      # 配置、鉴权、HTTP Server 等平台能力
├── internal/infrastructure/# MySQL 等基础设施实现
├── migrations/             # 数据库迁移占位
├── sql/                    # SQL 查询占位
├── .env.example            # 环境变量示例
├── go.mod
└── README.md
```

`internal/app/game` 仍保持同一个应用包，但已经开始按系统职责归口文件。建筑系统当前集中在：

- `building_registry.go`：建筑配置查询、建筑类型判断、核心建筑补齐。
- `building_lifecycle.go`：建筑状态变更和 `MutateBuilding` 统一入口。
- `building_effects.go`：建筑产量、容量和建筑 Modifier 来源。
- `service_building.go`：建筑升级、批量升级、极速完成等操作。

武将系统当前集中在：

- `general_registry.go`：武将配置查询、武将注册判断、全部武将配置列表。
- `general_growth.go`：武将等级、经验、属性点和战斗经验计算。
- `general_effects.go`：武将配置装配、属性拆解和 Modifier 来源。
- `general_combat.go`：武将特性接入战斗特性总线和战报结果。
- `service_general.go`：武将加点、洗点、换将等玩家操作。

兵种/军事系统当前集中在：

- `unit_registry.go`：兵种配置查询、兵种注册判断、按名称查找兵种。
- `army_state.go`：玩家兵力增加、兵力切片和 map 转换、战斗后兵力合并。
- `recruit_timing.go`：征兵时长、征兵消耗和征兵速度/消耗加成计算。
- `service_recruit.go`：征兵、极速完成征兵等玩家操作。
- `military_combat.go`：出兵扣兵、非战斗兵种限制、战斗单位构建和场景规则解析。

道具/Buff 系统当前集中在：

- `item_registry.go`：道具配置查询、道具注册判断、全部道具配置列表。
- `inventory_state.go`：玩家背包格子、堆叠拆分、聚合视图、道具增加和消耗。
- `drop_pools_config.go`：掉落池配置加载、查询和引用校验。
- `item_ledger.go`：物品流水模型和查询条件。
- `item_effects.go`：道具使用后的效果执行。
- `service_item.go`：道具发放、使用等玩家操作。
- `buff_effects.go`：Buff/DeBuff 状态、过期清理、Modifier scope 校验和 Modifier 来源。
- `service_buff.go`：Buff 发放、撤销等玩家操作。

玩法模块边界当前集中在：

- `gameplay_module_registry.go`：玩法模块边界声明，当前登记 `mail`、`minigame`、`reinforcement`、`pvp` 和 `reincarnation_abyss`。
- `service_mail.go`：信函列表、阅读、发送、删除和附件领取。
- `service_minigame.go`：万象幻境记录、鱼饵消耗、奖励兑换和兑换事件。
- `service_reinforcement.go`：增援派出、召回、遣返、到达、返程和战斗损耗接入。
- `service_reincarnation.go`：轮回绝境开启、18 波攻防结算、金币重置当前波随机加成、真实战损、累计奖励和副本战报接入。

后续活动/副本应先登记玩法模块边界，再通过奖励、事件、Modifier、建筑变更等核心入口接入长期资产。

Effect Pipeline 当前集中在：

- `internal/core/effect`：标准效果契约，只定义效果类型，不读写玩家存档。
- `effect_pipeline.go`：应用层效果执行器，复用奖励、建筑变更和 Buff/Modifier 入口。
- `item_effects.go`：道具效果已转换为标准 `reward` 效果。
- `building_lifecycle.go`：建筑变更已通过 `building_mutation` 效果执行。
- `service_mail.go`：信函附件已通过标准 `reward` 效果发放。
- `service_minigame.go`：万象幻境兑换已通过标准 `reward` 效果发放。

后续武将特性、活动、副本如果要影响长期资产，应优先提交标准 Effect。

## 本地运行

启动服务：

```bash
go run ./cmd/server
```

默认监听：

```text
http://localhost:8080
```

## 环境变量

可以参考 `.env.example`：

```text
HERO3_ENV=development
HERO3_PORT=8080
HERO3_VERSION=0.1.0
HERO3_LOG_LEVEL=info
HERO3_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174
# HERO3_DATABASE_DSN=hero3_user:hero3_password@tcp(127.0.0.1:3306)/hero3?parseTime=true&charset=utf8mb4&loc=UTC
# 本地开发建议使用 test_ 前缀测试库。
# HERO3_DATABASE_DSN=hero3_user:hero3_password@tcp(127.0.0.1:3306)/test_hero3?parseTime=true&charset=utf8mb4&loc=UTC
# HERO3_ALLOW_DEVELOPMENT_DATABASE=false
```

## 数据库

不配置 `HERO3_DATABASE_DSN` 时，服务使用内存存储，重启后账号和新建存档会丢失。

配置 `HERO3_DATABASE_DSN` 后，服务使用 MySQL/MariaDB，并在启动时自动创建当前需要的表：

- `accounts`：轻账号
- `players`：账号绑定的游戏存档，`state_json` 只保留轻量兼容快照，不再保存资源、建筑、兵力、武将、背包、Buff 等权威资产大字段
- `player_resources`：玩家资源权威表
- `player_inventory`：玩家背包权威表，按 `slot_id` 存储真实格子并兼容聚合视图
- `item_ledger`：物品获得和消耗流水
- `player_buildings`：玩家建筑权威表
- `player_resource_slots`：玩家资源田格子权威表
- `player_army_units` / `player_recruit_queues`：玩家兵力和征兵队列权威表
- `player_generals` / `player_general_assignments`：玩家武将和武将派驻权威表
- `player_buffs`：玩家 Buff/Modifier 权威表
- `reincarnation_runs` / `reincarnation_waves` / `reincarnation_battles`：轮回绝境副本实例、波次和战斗记录

本地或服务器 MySQL 可以先创建库和用户：

```sql
CREATE DATABASE hero3 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'hero3_user'@'%' IDENTIFIED BY 'hero3_password';
GRANT ALL PRIVILEGES ON hero3.* TO 'hero3_user'@'%';
FLUSH PRIVILEGES;
```

启动示例：

```bash
export HERO3_DATABASE_DSN='hero3_user:hero3_password@tcp(127.0.0.1:3306)/hero3?parseTime=true&charset=utf8mb4&loc=UTC'
go run ./cmd/server
```

当前开发约定：本地开发也使用服务器 MySQL，不再使用本机数据库。服务器已开放 MySQL `3306` 后，本地后端直接连接服务器数据库：

```bash
export HERO3_DATABASE_DSN='hero3_user:hero3_password@tcp(服务器IP:3306)/hero3?parseTime=true&charset=utf8mb4&loc=UTC'
go run ./cmd/server
```

项目根目录的 `dev.sh` 会自动读取 `go/.env`，本机可把实际 DSN 放在 `go/.env` 中。该文件已被 Git 忽略，不要提交数据库密码。

本地开发约定：

- 稳定玩家库使用 `hero3`。
- 本地开发库使用 `test_hero3`，避免开发调试误写稳定玩家数据。
- `HERO3_ENV=development` 时，后端默认拒绝连接非 `test_` 前缀数据库。
- 如确需临时连接非测试库，必须显式设置 `HERO3_ALLOW_DEVELOPMENT_DATABASE=true`。
- `make migrate` 只迁移当前 `HERO3_DATABASE_DSN` 指向的库。
- `make migrate-test` 会按当前库名生成 `test_` 前缀库并执行迁移；执行该命令的数据库账号必须拥有 `CREATE DATABASE` 权限。
- 如果数据库账号没有建库权限，需要先由服务器管理员创建并授权 `test_hero3`，再把 `go/.env` 的 DSN 改到该库。
- `make clone-data` 会从 `HERO3_SOURCE_DATABASE_DSN` 复制数据到当前 `HERO3_DATABASE_DSN` 指向的 `test_` 库，并清空目标库旧数据；复制完成后会自动回填并校验资源、背包、建筑、资源田格子、兵力、征兵队列、武将和 Buff 权威表。
- `clone-data` 支持目标测试库已经完成较新迁移的场景：目标表新增列会跳过并使用数据库默认值；源库存在但目标库缺失的列会中止复制。
- `clone-data --truncate` 会清空目标测试库全部普通表，再按源库表复制，避免目标新增表残留旧数据。
- `clone-data` 会跳过 `schema_migrations`，避免把源库迁移历史覆盖到测试库。

数据库维护命令：

```bash
make migrate
make migrate-test
make print-test-dsn
HERO3_SOURCE_DATABASE_DSN='源库DSN' make clone-data
make backfill-resources
make verify-resources
make backfill-inventory
make verify-inventory
make backfill-buildings
make verify-buildings
make backfill-resource-slots
make verify-resource-slots
make backfill-army
make verify-army
make backfill-recruit-queues
make verify-recruit-queues
make backfill-generals
make verify-generals
make backfill-buffs
make verify-buffs
make healthcheck-authority
```

`verify-*` 命令用于旧 `state_json` 快照迁移期的一致性校验；当前日常开发以 `make healthcheck-authority` 判断权威表是否完整、`players.state_json` 是否仍残留资源、背包、建筑、资源田、兵力、招募队列、武将、Buff 等大字段。

日常开发只需要：

```bash
./dev.sh
```

如果后续关闭公网 `3306`，可以把 `go/.env` 的 `HERO3_DB_TUNNEL_ENABLED` 改为 `true`，让 `dev.sh` 自动启动 SSH 隧道。

## 在线接口文档

Go 后端启动后会同时提供 Scalar 在线接口文档：

```text
http://localhost:8080/docs
```

文档页面读取：

```text
http://localhost:8080/openapi.yaml
```

默认读取项目根目录的 `docs/接口文档/openapi打包.yaml`。如果启动目录特殊，或者需要临时读取其他文件，可以设置：

```bash
export HERO3_OPENAPI_PATH='../docs/接口文档/openapi打包.yaml'
```

每次新增或修改接口后，先在项目根目录运行：

```bash
make openapi
```

再刷新 `/docs` 页面查看最新接口。

## 基础接口

健康检查：

```bash
curl http://localhost:8080/healthz
```

服务信息：

```bash
curl http://localhost:8080/api/v1/meta
```

游戏启动信息：

```bash
curl http://localhost:8080/api/v1/game/bootstrap
```

游戏状态快照：

```bash
curl http://localhost:8080/api/v1/game/state
```

注册轻账号：

```bash
curl -X POST http://localhost:8080/api/v1/accounts/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"123456"}'
```

登录轻账号：

```bash
curl -X POST http://localhost:8080/api/v1/accounts/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"123456"}'
```

查看账号存档：

```bash
curl http://localhost:8080/api/v1/accounts/{accountId}/players
```

创建账号绑定存档：

```bash
curl -X POST http://localhost:8080/api/v1/players/create \
  -H 'Content-Type: application/json' \
  -d '{"accountId":"acc_xxx","nickname":"主公","faction":"wei"}'
```

`generalId` 可选；不传时后端会选择该阵营第一个可用将领。前端选择角色页会显式传入玩家选中的将领。

## 开发约定

- 页面展示状态放在前端，核心游戏数据以后端为准。
- 战斗、资源结算、存档版本迁移等关键逻辑应放在 Go 后端。
- 新业务先判断归属：HTTP 输入输出放 `internal/transport`，业务编排放 `internal/app`，可复用领域能力放 `internal/core`，数据库和外部设施放 `internal/infrastructure`。
- HTTP 层只做参数解析、权限校验、服务调用和响应封装。
- 数据库结构确定前，`migrations/` 与 `sql/` 先保留为空目录。
