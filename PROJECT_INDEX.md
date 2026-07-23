<!-- 本文件是 Hero3 面向 AI 与新协作者的项目导航索引，用于快速定位权威规则、代码入口和验证命令。 -->

# Hero3 AI 项目索引

最后核对：2026-07-23
适用分支：`main-core`

## 1. 这个索引怎么用

本文件是“导航地图”，不是新的业务规则或实现事实来源。AI 接手任务时按以下顺序建立上下文：

1. 读 `AGENTS.md`，确认执行、记忆和提交规则。
2. 读 `PROJECT_RULES.md`，确认稳定核心、模块边界和生产约束。
3. 读 `memory/` 中日期最新的文件，了解最近变更。
4. 读本文件，定位本次任务涉及的代码、文档和测试。
5. 打开相关源文件、OpenAPI 和专题文档，以当前代码验证索引中的描述。

发生冲突时，按 `PROJECT_RULES.md` 规定的优先级判断：

1. `PROJECT_RULES.md`
2. `AGENTS.md`
3. 最近的 `memory/YYYY-MM-DD.md`
4. 当前代码和配置
5. `docs/` 中仍保留的当前文档

已删除的历史文档不作为开发依据；如需追溯只能查看 Git 历史，并用当前权威来源重新核验。`PROJECT_INDEX.md` 只帮助定位，不能覆盖以上任何权威来源。

## 2. 一分钟项目画像

Hero3 是一个三国题材策略网页游戏，后端采用“稳定核心 + 可扩展玩法模块”的模块化单体架构，仓库同时维护两个玩家前端和一个 GM 后台。

| 运行单元 | 技术 | 目录 | 入口 | 生产用途 |
| --- | --- | --- | --- | --- |
| Go API | Go 1.26、`net/http`、MySQL | `go/` | `go/cmd/server/main.go` | 账号、存档、资产、战斗、玩法、配置与运营接口 |
| 现代玩家端 | React 19、TypeScript、Vite、Zustand | `web/` | `web/src/main.tsx`、`web/src/App.tsx` | `hero3.ccoos.cn` |
| 武林三国风格玩家端 | Vue 3、TypeScript、Vite | `web-wlsg/` | `web-wlsg/src/main.ts`、`web-wlsg/src/App.vue` | `wlsg.ccoos.cn`，与 React 端并存 |
| GM 后台 | React 19、TypeScript、Vite | `admin/` | `admin/src/main.tsx`、`admin/src/App.tsx` | 线上运营、配置与排障 |
| OpenAPI | 拆分 YAML | `docs/接口文档/openapi/` | `openapi/openapi.yaml` | 接口契约；打包结果为 `openapi打包.yaml` |
| 玩家帮助内容 | Markdown | `helpdocs/content/` | `go/internal/transport/helpdocs/handler.go` | 当前仅保留空目录，旧正文已清理 |
| 运维与发布 | GitHub Actions、Nginx、systemd | `.github/workflows/`、`deploy/` | `.github/workflows/deploy.yml` | `main-core` 推送后由服务器拉源码并构建部署 |

三套前端共享同一个 Go API。两个玩家端不是替换关系；任务必须先确认目标是 `web/` 还是 `web-wlsg/`。进入 `web-wlsg/` 开发前还必须读取 `web-wlsg/AGENTS.md`。

## 3. 总体数据流

```text
React web / Vue web-wlsg / React admin
                  |
                  v
        transport/api/router.go
                  |
                  v
       transport/api/handlers_*.go
                  |
                  v
       app/game/service_*.go
          |               |
          v               v
 app/game 核心资产编排   internal/core 纯领域能力
          |
          v
      Repository 接口
          |
          v
 infrastructure/storage/mysql_*.go
          |
          v
         MySQL
```

HTTP 层只负责输入、鉴权、调用和输出；玩法规则与事务编排放在 `internal/app/game`；可复用纯领域能力放在 `internal/core`；MySQL 实现放在 `internal/infrastructure/storage`。

## 4. 根目录导航

| 路径 | 定位 | 何时阅读 |
| --- | --- | --- |
| `AGENTS.md` | AI 执行规则、中文回复、记忆与提交要求 | 每次新会话首先阅读 |
| `PROJECT_RULES.md` | 长期架构和产品判断规则 | 所有架构、模块、数据库和实现判断前 |
| `PROJECT_INDEX.md` | 本导航索引 | 定位代码与验证范围时 |
| `UI_RULES.md` | Hero3 水墨淡彩国风美术规则 | 生成素材或修改 React 玩家端视觉时 |
| `README.md` | 项目运行、部署、当前能力概览 | 初次了解和运行项目时 |
| `Makefile` | 跨子项目构建、OpenAPI、数据库工具入口 | 查找统一命令时 |
| `dev.sh` / `dev.ps1` / `dev.bat` | 本地多服务启动 | 联调时 |
| `go/` | 后端、配置、数据库工具 | 服务端和数据任务 |
| `web/` | 现代 React 玩家端 | `hero3.ccoos.cn` 页面任务 |
| `web-wlsg/` | 武林三国风格 Vue 玩家端 | `wlsg.ccoos.cn` 页面任务 |
| `admin/` | GM 管理端 | 线上运营与配置界面任务 |
| `docs/` | 当前设计、数据库、接口、测试和部署文档 | 领域设计和契约对齐 |
| `helpdocs/` | 玩家帮助页内容源 | 玩家帮助任务 |
| `scripts/` | OpenAPI 打包、压测等工具 | 生成文档或专项验证 |
| `deploy/` | Nginx、systemd、journald 配置 | 发布和线上运维任务 |
| `.github/workflows/` | 生产自动部署调度 | 修改 CI/CD 时 |
| `announcements/daily/` | 玩家可读每日更新公告 | 准备上线公告时 |
| `memory/` | 按日期记录的开发记忆 | 新会话、提交前和追溯近期决策时 |

## 5. Go 后端索引

### 5.1 启动与技术平台

| 职责 | 入口 |
| --- | --- |
| 服务启动、配置装载、仓储选择、黄巾调度器 | `go/cmd/server/main.go` |
| 数据库维护、迁移、回填、清理、巡检 | `go/cmd/dbtool/main.go` 与同目录子命令文件 |
| 环境变量与默认配置路径 | `go/internal/platform/config/config.go`、`go/.env.example` |
| JWT、玩家归属、Admin Token | `go/internal/platform/auth/auth.go` |
| HTTP Server 超时与生命周期 | `go/internal/platform/httpserver/server.go` |
| API 总路由与公开路径白名单 | `go/internal/transport/api/router.go` |
| 在线 OpenAPI 文档 | `go/internal/transport/apidocs/docs.go` |
| 玩家帮助文档接口 | `go/internal/transport/helpdocs/handler.go` |

后端未设置 `HERO3_DATABASE_DSN` 时使用内存仓储；设置后使用 MySQL。开发环境默认只允许连接 `test_` 前缀数据库，生产迁移和回填属于高风险操作，必须遵守 `PROJECT_RULES.md` 的确认要求。

### 5.2 应用层文件命名规律

`go/internal/app/game/` 当前仍是单一 Go package，按以下规则快速定位：

| 文件模式 | 内容 |
| --- | --- |
| `service.go` | `Service` 聚合、内存仓储初始化和通用入口 |
| `service_<domain>.go` | 某领域的用例编排和事务入口 |
| `<domain>.go` | 领域模型、规则和辅助算法 |
| `<domain>_config.go` | JSON/数据库配置结构与加载 |
| `<domain>_registry.go` | 可注册内容查询与校验 |
| `repository*.go` | 仓储端口；实现位于 `infrastructure/storage` |
| `model.go` | 账号、玩家、`GameState` 等兼容聚合模型 |
| `reward.go` | 标准奖励应用和副作用编排 |
| `event.go`、`core_asset_events.go` | 应用事件与核心资产事件发布 |
| `core_resource_transaction.go` | 资源长期资产事务入口 |
| `gameplay_module_registry.go` | 已登记玩法模块的边界、状态归属和核心接入点 |

### 5.3 稳定核心能力

| 核心能力 | 应用层定位 | 纯核心定位 | 存储定位 |
| --- | --- | --- | --- |
| 玩家状态与资产事务 | `repository.go`、`core_resource_transaction.go`、`service_views.go` | — | `mysql_player_state.go`、各 `mysql_*_assets.go` |
| 资源与货币 | `service_resource.go`、`service_gold.go`、`gold_ledger.go` | `core/reward/` | `mysql_player_state.go`、`mysql_reward_assets.go`、`mysql_currency.go`、`mysql_ledger.go` |
| 建筑 | `building_*.go`、`service_building.go` | `core/building/` | `mysql_buildings.go`、`mysql_building_assets.go` |
| 兵力与征兵 | `army_state.go`、`unit_registry.go`、`recruit_timing.go`、`service_recruit.go` | — | `mysql_army.go`、`mysql_recruit_assets.go` |
| 武将与特性 | `general_*.go`、`service_general.go` | `core/general/`、`core/general/traits/` | `mysql_generals.go`、`mysql_general_assets.go` |
| 物品与背包 | `inventory_state.go`、`item_*.go`、`service_item.go` | `core/reward/`、`core/registry/` | `mysql_item_assets.go`、`mysql_ledger.go` |
| Buff 与 Modifier | `buff_effects.go`、`modifier.go`、`effect_pipeline.go` | `core/effect/`、`core/modifier/` | `mysql_buffs.go` |
| 战斗引擎 | `military_combat.go`、`general_combat.go`、`service_combat.go` | `core/combat/` | `mysql_combat_assets.go` |
| 奖励与事件 | `reward.go`、`event.go`、`core_asset_events.go` | `core/reward/`、`core/event/` | 由各组合事务和流水实现承接 |
| 内容注册 | `registry.go`、各 `*_registry.go` | `core/registry/` | 配置文件或数据库配置 |

玩家长期资产不能由玩法模块直接改 `GameState` 或绕过事务落库；新增玩法先回答读取、修改、锁定、奖励、事件和模块状态六个问题。

### 5.4 玩法与业务模块

| 模块 | 应用层 | Handler | MySQL | 配置/设计 |
| --- | --- | --- | --- | --- |
| NPC 城池与扫荡 | `npc.go`、`service_npc.go`、`sweep_task.go`、`service_sweep_task.go` | `handlers_map.go` | `mysql_npc_state.go`、`mysql_sweep_task.go` | `go/config/npc.json`、`docs/系统设计/NPC城池系统设计.md` |
| PVP | `pvp.go`、`service_pvp.go` | `handlers_pvp.go`、`handlers_admin_pvp.go` | `mysql_pvp.go` | `docs/系统设计/PVP系统开发文档.md` |
| 增援 | `reinforcement.go`、`service_reinforcement.go` | `handlers_reinforcements.go` | `mysql_reinforcements.go` | `docs/系统设计/增援系统开发文档.md` |
| 轮回绝境 | `reincarnation*.go`、`service_reincarnation.go` | `handlers_reincarnation.go`、`handlers_admin_reincarnation.go` | `mysql_reincarnation.go` | `go/config/reincarnation.json`、对应系统设计文档 |
| 万象幻境 | `service_minigame.go`、`fishing_config.go`、`slot_config.go` | `handlers_minigame.go` | `mysql_minigame.go` | `fishing.json`、`slot.json`、小游戏系统设计文档 |
| 黄巾起义 | `yellow_turban*.go`、`service_yellow_turban.go` | `handlers_yellow_turban.go` | `mysql_yellow_turban.go` | `yellow_turban.json`、`docs/系统设计/黄巾起义最终开发文档.md` |
| 世界地图 | `world_map.go`、`service_world_map.go` | `handlers_world_map.go` | `mysql_world_map.go` | 世界地图系统设计文档 |
| 战报 | `report_standard.go`、各战斗 service | `handlers_reports.go`、`handlers_admin_reports.go` | `mysql_report.go`、`mysql_events.go` | 战报设计文档、`schemas/report.yaml` |
| 信函 | `service_mail.go` | `handlers_mail.go` | `mysql_mail.go` | `schemas/mail.yaml` |
| 公告 | `announcement.go`、`service_announcement.go` | `handlers_announcement.go`、`handlers_admin_announcement.go` | `mysql_announcement.go` | 公告系统设计文档 |

`gameplay_module_registry.go` 当前正式登记 `mail`、`minigame`、`reinforcement`、`pvp` 和 `reincarnation_abyss`；新增正式玩法时应同步检查是否需要登记边界声明。

### 5.5 配置索引

| 配置 | 用途 |
| --- | --- |
| `go/config/balance.json` | 建筑、资源、加速与基础平衡 |
| `go/config/factions.json` | 阵营信息 |
| `go/config/units/{wei,shu,wu}.json` | 各阵营兵种 |
| `go/config/generals.json` | 武将与双特性 |
| `go/config/combat.json` | 战斗规则、场景映射与城墙系数 |
| `go/config/items.json` | 物品注册和效果 |
| `go/config/drop_pools.json` | 掉落池 |
| `go/config/npc.json` | NPC 城池 |
| `go/config/fishing.json` | 仙池垂钓默认模板 |
| `go/config/slot.json` | 天机轮转 |
| `go/config/reincarnation.json` | 轮回绝境 |
| `go/config/yellow_turban.json` | 黄巾起义默认配置 |

线上 GM 可修改配置的权威来源应是 `game_configs` 表；JSON 主要是默认模板、字段结构、开发兜底和初始化种子。修改配置加载逻辑时同时检查 `service_game_config.go`、`repository_game_config.go` 与 `mysql_game_config.go`。

### 5.6 数据库定位

- 仓储接口：`go/internal/app/game/repository*.go`。
- MySQL 主实现：`go/internal/infrastructure/storage/mysql*.go`。
- 初始化与兼容迁移：主要位于 `mysql.go` 及相关存储文件。
- 数据库命令：`go/cmd/dbtool/`，根 `Makefile` 提供常用别名。
- 结构说明：`docs/数据库/数据库设计.md`。
- 状态读写约束：`docs/架构/玩家状态读写模型规范.md`。
- `go/migrations/` 与 `go/sql/` 当前只是占位，不能据此误判为没有数据库结构。

高风险提示：生产数据库、回填、清理、迁移和发布操作不能因为存在 Make target 就直接执行。

## 6. API 契约索引

同一个接口通常要同时核对以下位置：

```text
go/internal/transport/api/router.go
  -> handlers_<domain>.go
  -> app/game/service_<domain>.go
  -> repository*.go / mysql_*.go
  -> docs/接口文档/openapi/paths/<domain>.yaml
  -> docs/接口文档/openapi/schemas/<domain>.yaml
  -> web/src/api/game.ts 或 web-wlsg/src/api/gameApi.ts 或 admin/src/api/admin.ts
```

接口变更后运行 `make openapi`，它会校验拆分文档并更新 `docs/接口文档/openapi打包.yaml`。路由的运行时事实以 `router.go` 为准，接口请求/响应契约需要与 OpenAPI 和调用方保持同步。

主要路径分组：

| 分组 | OpenAPI paths 文件 | 后端路由注册函数 |
| --- | --- | --- |
| 账号与存档 | `paths/account.yaml` | `registerAccountRoutes` |
| 游戏状态 | `paths/game.yaml` | `registerPublicRoutes` |
| 城池资源 | `paths/city.yaml` | `registerCityRoutes` |
| 军事武将 | `paths/military.yaml` | `registerMilitaryRoutes` |
| 地图、NPC、世界地图、黄巾 | `paths/map.yaml` | `registerMapRoutes` |
| PVP | 当前玩家与 GM PVP 路由尚未完整进入 OpenAPI，是待补契约缺口 | `registerPvpRoutes`、Admin 路由 |
| 增援 | `paths/reinforcement.yaml` | `registerReinforcementRoutes` |
| 副本 | `paths/dungeon.yaml` | `registerReincarnationRoutes` |
| 战报 | `paths/reports.yaml`、`paths/news.yaml` | `registerReportRoutes` |
| 信函 | `paths/mail.yaml` | `registerMailRoutes` |
| 公告 | `paths/announcement.yaml` | `registerAnnouncementRoutes` |
| 物品 | `paths/items.yaml` | `registerItemRoutes` |
| 小游戏 | `paths/minigame.yaml` | `registerMiniGameRoutes` |
| GM | `paths/admin.yaml` | `registerAdminRoutes` |

## 7. 前端索引

### 7.1 React 玩家端 `web/`

| 职责 | 入口 |
| --- | --- |
| 应用启动与路由 | `web/src/main.tsx`、`web/src/App.tsx` |
| 统一请求、鉴权与错误映射 | `web/src/api/client.ts` |
| 游戏接口 | `web/src/api/game.ts` |
| 服务端契约类型 | `web/src/types/game.ts` |
| 账号会话 | `web/src/store/accountStore.ts` |
| 启动配置 | `web/src/store/configStore.ts` |
| 玩家权威状态 | `web/src/store/gameStore.ts` |
| 公共布局 | `web/src/components/Layout.tsx` |
| 页面 | `web/src/pages/{city,military,map,news,mail,notice,help,...}` |
| 测试 | `web/tests/` 与页面同目录逻辑测试 |

主要路由在 `web/src/App.tsx`：`/city`、`/military`、`/map`、`/alliance`、`/news`、`/mail`、`/notice`、`/help`、`/settings`、`/account`；`/report/:reportId` 是分享战报入口。

### 7.2 Vue 玩家端 `web-wlsg/`

该目录有额外强约束，任何修改前先读 `web-wlsg/AGENTS.md`、`web-wlsg/docs/开发标准/官方页面复刻标准.md` 和 `web-wlsg/docs/开发标准/现有后端API接入标准.md` 中与任务相关的部分。

| 职责 | 入口 |
| --- | --- |
| 应用状态入口 | `web-wlsg/src/App.vue` |
| 主界面外壳与页面切换 | `web-wlsg/src/components/GameShell.vue` |
| 请求客户端与 API | `web-wlsg/src/api/client.ts`、`web-wlsg/src/api/gameApi.ts` |
| 登录选档 | `web-wlsg/src/session/` |
| 城池、军事与行军状态 | `web-wlsg/src/game/` |
| 世界地图 | `web-wlsg/src/worldMap/`、`MapStage.vue` |
| NPC | `web-wlsg/src/npc/`、`NpcDirectory.vue` |
| 军情战报 | `web-wlsg/src/intelligence/`、`IntelligenceStage.vue` |
| 轮回绝境 | `web-wlsg/src/dungeon/`、`DungeonStage.vue` |
| 万象幻境 | `web-wlsg/src/mirage/`、`MirageStage.vue` |
| 官方复刻素材 | `web-wlsg/public/assets/official/` |
| Vitest 测试 | 各领域目录的 `*.test.ts` |

这个前端不使用前端伪数据推演结果，所有资产、战斗、行军和奖励以现有后端响应为准。

### 7.3 GM 后台 `admin/`

| 职责 | 入口 |
| --- | --- |
| 应用与页面切换 | `admin/src/App.tsx` |
| 统一 GM API | `admin/src/api/admin.ts` |
| 类型 | `admin/src/types/` |
| 数据聚合 | `admin/src/hooks/useAdminDashboard.ts` |
| 功能面板 | `admin/src/components/*Panel.tsx` |

后台通过 `X-Admin-Token` 调用 Admin 接口。用户要求发布公告、发放资产或调整 GM 配置时，默认目标是线上 `https://hero3.ccoos.cn`，不能用本地成功代替线上操作成功。

## 8. 文档索引

文档总入口是 `docs/文档目录.md`。按任务优先读取：

| 任务 | 首选文档 |
| --- | --- |
| 架构与模块边界 | `PROJECT_RULES.md`、`docs/架构/总体架构设计.md`、`docs/架构/核心地基设计.md` |
| 玩家长期资产和事务 | `docs/架构/玩家状态读写模型规范.md` |
| 数据库表与索引 | `docs/数据库/数据库设计.md` |
| API | `docs/接口文档/OpenAPI维护规范.md`、拆分 OpenAPI |
| 认证与权限 | `docs/安全与权限/安全权限规范.md` |
| 测试范围 | `docs/测试/测试策略.md` |
| 发布与回滚 | `docs/发布与回滚/发布回滚规范.md`、`docs/运维部署/服务器部署文档.md` |
| 某个玩法 | `docs/系统设计/` 下同名文档，同时以代码核验当前完成度 |
| 曹操特性 | `docs/系统设计/曹操特性更新开发文档.md`、`go/config/generals.json`、两端 `guardProjection.ts` |
| 甄宓特性 | `docs/系统设计/甄宓特性更新开发文档.md`、`go/config/generals.json`、`go/internal/core/general/traits/catalog.go`、两端战报适配器 |
| 司马懿特性 | `docs/系统设计/司马懿特性更新开发文档.md`、`go/config/generals.json`、`go/internal/core/general/traits/catalog.go`、PVP/NPC/黄巾/援军测试、两端战报适配器 |
| 夏侯渊特性 | `docs/系统设计/夏侯渊特性更新开发文档.md`、`go/config/generals.json`、被动兵种 Modifier、PVP/黄巾/援军测试、两端战报被动与触发分栏 |
| 张辽特性 | `docs/系统设计/张辽特性更新开发文档.md`、`go/config/generals.json`、战前溃逃适配、PVP/NPC/轮回/方向测试、两端战报适配器 |
| 许褚特性 | `docs/系统设计/许褚特性更新开发文档.md`、`go/config/generals.json`、兵种被动 Modifier、PVP/NPC/方向测试、两端战报被动与触发分栏 |
| 典韦特性 | `docs/系统设计/典韦特性更新开发文档.md`、`go/config/generals.json`、战前攻防修正、PVP/NPC/黄巾/增援测试、两端战报适配器 |
| 产品方向 | `docs/产品/项目最终目的.md`、`docs/产品/未来开发规划.md` |

专题设计文档可能同时包含已完成事实和后续规划，读取后必须用当前代码、OpenAPI 和最近记忆核验。

当前已知的导航缺口：PVP 的运行时路由已在 `router.go` 注册并被前端使用，但拆分 OpenAPI 尚未完整覆盖玩家和 GM PVP 接口；涉及 PVP 时必须直接核对路由、Handler、前端类型与系统设计，不能误以为 OpenAPI 已完整描述。

## 9. 按任务快速定位

| 要做什么 | 先看哪里 | 通常还要同步 |
| --- | --- | --- |
| 新增或修改 API | `router.go`、对应 handler/service/repository | OpenAPI、目标前端 API、类型、测试、README |
| 修改长期资产 | `PROJECT_RULES.md`、资产事务与奖励入口 | MySQL 权威表、流水、事件、并发测试 |
| 新增玩法模块 | `gameplay_module_registry.go`、核心接入问题 | 模块状态仓储、奖励、事件、OpenAPI、设计文档 |
| 修改战斗 | `core/combat/`、`general_combat.go`、相关玩法 service | 战报快照、脱敏、双方资产事务、测试 |
| 修改战报 | `report_standard.go`、`mysql_report.go`、handlers | OpenAPI report schema、两个玩家端、GM 排查页 |
| 修改配置 | 对应 JSON 和 `*_config.go` | `game_configs` 读写、GM 面板、校验、默认种子 |
| 修改 React 玩家端 | `web/src/App.tsx`、目标 page/store/API | `web/README.md`、构建和相关测试 |
| 修改武林三国前端 | `web-wlsg/AGENTS.md`、目标 state/component | `web-wlsg/README.md`、Vitest、构建、视觉验收 |
| 修改 GM 后台 | `admin/src/App.tsx`、`api/admin.ts`、目标 Panel | 后端 Admin 路由、权限、二次确认、README |
| 修改数据库 | repository 端口与 `mysql_*.go` | 数据库设计、dbtool、回填/回滚/校验方案 |
| 修改部署 | `.github/workflows/deploy.yml`、`deploy/` | 发布回滚文档、双域名和健康检查 |
| 提交或推送 | `git diff`、当天 `memory/YYYY-MM-DD.md` | 中文 README/docs、测试结果、远端 `main-core` |

## 10. 常用检索命令

```bash
# 找路由与接口处理器
rg -n 'HandleFunc|register[A-Za-z]+Routes' go/internal/transport/api

# 找某个业务词在服务、存储、前端和文档中的完整链路
rg -n '关键词' go/internal/app go/internal/infrastructure web/src web-wlsg/src admin/src docs

# 找仓储端口及其 MySQL 实现
rg -n 'type .*Repository interface|func \(r \*MySQLRepository\)' go/internal/app/game go/internal/infrastructure/storage

# 找玩家状态写入和组合事务
rg -n 'Update(Player|Account|Mail|MiniGame|Pvp|Reincarnation)|Transaction' go/internal/app/game go/internal/infrastructure/storage

# 找标准奖励、事件与 Modifier
rg -n 'GrantRewards|ApplyRewards|Publish|Modifier' go/internal/app/game go/internal/core

# 找某 API 在三套前端中的调用
rg -n '/api/v1|/city/|/pvp/|/reports/' web/src web-wlsg/src admin/src

# 查看当前有效文档与开发记忆
rg -n '关键词' PROJECT_RULES.md AGENTS.md README.md docs helpdocs memory
```

不要从 `public/assets/official/`、构建产物、二进制或 Playwright 临时输出开始理解业务，它们会制造大量噪音。Git 历史中的已删除文档同样不能作为现行事实。

## 11. 构建与验证矩阵

| 范围 | 命令 |
| --- | --- |
| 全部生产构建 | `make build` |
| Go 全量测试 | `cd go && go test ./...` |
| Go 服务构建 | `make build-go` |
| 数据库工具构建 | `make build-dbtool` |
| React 玩家端 | `cd web && pnpm test && pnpm lint && pnpm build` |
| Vue 玩家端 | `cd web-wlsg && pnpm test && pnpm build` |
| GM 后台 | `cd admin && pnpm lint && pnpm build` |
| OpenAPI | `make openapi` |
| 差异格式检查 | `git diff --check` |

验证范围应与变更风险匹配。数据库相关测试可能要求显式测试 DSN；不得默认连接生产库。涉及 `web-wlsg` UI 时，还要按其目录规则执行浏览器控制台、目标视口和截图验收。

## 12. AI 修改前检查清单

- 已确认目标前端是 `web/` 还是 `web-wlsg/`。
- 已区分技术核心、游戏核心和玩法模块。
- 已识别会读取、修改或锁定哪些长期资产。
- 已找到 Handler → Service → Repository → MySQL → 前端的完整链路。
- 已检查对应 OpenAPI、配置和专题文档。
- 未从历史文档或前端展示反推服务端事实。
- 未让玩法绕过核心事务、奖励、事件或注册表。
- 涉及生产、迁移、回填、清理时已停下来取得用户明确确认。
- 完成后会同步 README/docs、运行相关测试，并在提交前更新当天记忆。

## 13. 索引维护规则

出现以下变化时同步更新本文件：

- 新增或删除顶层运行单元。
- 前端入口、后端分层或主路由发生变化。
- 新增正式玩法模块或核心资产入口。
- 权威配置、数据库结构位置或部署方式变化。
- 文档权威入口和验证命令变化。

只记录稳定的导航信息，不在本文件复制完整需求、接口字段、数据库表结构或阶段进度；这些内容应留在代码、OpenAPI、专题文档和 `memory/` 中。
