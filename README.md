# Hero3

`Hero3` 是一个三国题材策略网页游戏项目。当前项目采用前后端分离结构，前端负责页面展示与交互，后端负责数据接口、存档、战斗结算和后续核心玩法规则。

## 项目结构

```text
Hero3/
├── web/    # React + TypeScript + Vite 玩家前端
├── admin/  # React + TypeScript + Vite GM 后台
├── go/     # Go 后端 API 服务
└── helpdocs/ # 玩家端帮助文档站内容
```

Go 后端当前采用模块化单体目录：

```text
go/internal/
├── transport/       # HTTP/API 输入输出层
├── app/             # 应用服务与业务编排
├── core/            # 通用领域能力，如战斗、将领事件
├── platform/        # 鉴权、配置、HTTP Server 等平台能力
└── infrastructure/  # MySQL 等基础设施实现
```

## 前端

前端目录：

```bash
cd web
```

常用命令：

```bash
pnpm install
pnpm dev
pnpm build
pnpm lint
```

## 后端

后端目录：

```bash
cd go
```

常用命令：

```bash
go run ./cmd/server
go test ./...
go build ./cmd/server
```

数据库：

- 默认不配置数据库时使用内存存储，适合快速开发。
- 配置 `HERO3_DATABASE_DSN` 后启用 MySQL/MariaDB；开发环境默认随服务启动执行轻量迁移，生产环境默认跳过启动迁移，结构变更应在低峰通过 `make migrate` 或 `hero3-dbtool migrate` 执行。
- `game_configs` 保存线上 GM 配置，配置文件只作为默认模板和数据库缺失时的初始化种子；当前仙池垂钓配置已使用 `game_configs.fishing`，GM 后台保存不会回写发布目录 JSON。
- 本地开发模式应连接 `test_` 前缀测试库，例如 `test_hero3`，不要直接写稳定玩家库。
- `make migrate` 迁移当前 DSN 指向的库，`make migrate-test` 创建并迁移 `test_` 前缀测试库。
- `make clone-data` 可从 `HERO3_SOURCE_DATABASE_DSN` 复制数据到当前 `test_` 目标库，复制后自动回填并校验资源、背包、建筑、资源田格子、兵力、征兵队列、武将、Buff、玩家货币和旧 NPC 状态权威表。
- `make healthcheck-authority` 是当前权威表健康检查；`verify-*` 仍保留为旧快照迁移期校验，不作为轻量 `state_json` 后的日常通过标准。
- `make backfill-currencies` / `make verify-currencies` 和 `make backfill-npc-states` / `make verify-npc-states` 用于把旧玩家的城金、兑换冷却和旧 NPC 快照迁移到新权威表；正式库执行回填需直接调用 dbtool 并显式传 `--allow-non-test`，回填按小批次提交，可重复执行。
- `make backfill-world-positions` 用于在测试库为老玩家补齐世界地图权威坐标；正式库执行需直接调用 `hero3-dbtool backfill-world-positions --allow-non-test --batch-size=500`，命令只分页扫描缺坐标的 `players.id`，不读取大存档、不复制大表，可重复执行，已有坐标会跳过，输出创建、跳过、冲突、失败和剩余数量；线上试跑可加 `--max-batches=1`。
- `make report-stats`、`make lock-snapshot` 和 `make cleanup-battle-reports-dry-run` 用于战报增长、锁等待和过期战报清理观测；生产部署会安装 `hero3-dbtool`，并启用战报清理、每小时维护巡检和每日更新公告 systemd timer。
- `hero3-dbtool publish-daily-update-announcement` 会按 `/opt/hero3/RELEASE`、`/opt/hero3/source` 的提交记录和当天 `memory/YYYY-MM-DD.md` 生成“每日更新公告”，默认 dry-run；生产定时任务每天 23:50 执行 `--execute --allow-non-test`，发布后用 `game_configs.daily_update_announcement_cursor` 记录已公告到的提交，避免重复发布。
- 战报清理默认按差异化保留执行：普通/NPC/扫荡 72 小时，PVP/玩家城池来源、防守/侦查 168 小时，玩家软删除 24 小时；有效分享链接会被保护。
- `make ensure-report-cleanup-indexes-dry-run` 用于检查战报清理和可见上限所需索引；正式库创建缺失索引需直接调用 dbtool 并显式传 `--execute --allow-non-test`，建议低峰执行。
- `make maintenance-status` 是只读维护巡检汇总，会同时检查战报统计、战报生命周期索引、可清理候选量和权威表健康；缺索引时会跳过候选量统计并返回异常。生产环境每小时自动执行一次并写入 systemd journal。
- `clone-data` 允许目标测试库比源库多出迁移后的新列，复制时只写公共列；源库列在目标库不存在时会中止。
- `clone-data` 不复制或清空 `schema_migrations`，测试库迁移记录由测试库自己的迁移命令维护。
- 物品系统使用 `go/config/items.json` 和 `go/config/drop_pools.json` 配置注册；背包权威表按格子 `slot_id` 存储，兼容接口仍返回按物品聚合的 `inventory`。
- NPC 层级可在 `go/config/npc.json` 通过 `dropPoolId` 绑定掉落池；掉落池支持 `slots` 独立槽位和 `none` 空掉落，用于配置保底、低概率和多段概率奖励。
- 万象幻境当前包含仙池垂钓、军营豪赌和天机轮转；军营豪赌与天机轮转均由后端结算并复用 `minigame_records` 库存兑换体系。仙池垂钓的文件配置只作为模板，线上 GM 修改写入数据库 `game_configs.fishing`，避免发布覆盖运营配置；钓鱼鱼饵支持独立 `rarityWeights`、最低/最高品质、神话 `mythic` 池和咬钩等待倍率，旧 `rarityBoost` 仅作为兼容兜底。天机轮转配置仍在 `go/config/slot.json`，GM 后台可调整每线押注、图案权重、倍率、免费旋转和宝匣倍率。天机轮转第二版采用每线押注、固定 5 线、3x3 服务端结果矩阵，并支持 Wild、Scatter 免费旋转和 Bonus 奖励。
- 物品获得和消耗会写入 `item_ledger`，GM 后台可查看物品配置、玩家背包格子和物品流水。
- 战斗规则使用 `go/config/combat.json`，GM 后台可调整损失指数、场景规则映射、悬殊战力无损阈值 `noLossPowerRatioThreshold`、阵营城墙系数和城墙硬度预留参数；PVP 守城按阵营城墙系数 `base^城墙等级` 提高防御。
- 购买产量/容量加成时，同倍率续订只叠加剩余时间；不同倍率购买会按新倍率和新时长重新计算。
- 轮回绝境副本使用 `go/config/reincarnation.json` 配置层级、波次、加成、金币重置随机加成价格和奖励；玩家入口位于“地图 -> 副本”，GM 后台可编辑 JSON 配置、查看实例并处理异常结算。
- NPC、PVP 和副本等真实战斗统一使用“参战武将”规则：每次最多携带 1 名武将；出征或副本波次显式携带武将时才享受武将加成、触发对应战斗特性并在杀敌后获得经验；不带武将时只享受玩家自身基础加成。玩家被攻击时，只有留在家中且未出征/增援的主将参与守城加成、战斗特性和经验。
- 将领系统采用双特性结构：每个启用将领必须同时配置 `specialTrait` 特殊特性和 `bonusTrait` 加成特性；当前 `go/config/generals.json` 覆盖全部现有将领，图片中的百分比、固定值、上限和作用范围均进入 GM 可配置参数，不按 `rarity` 决定强弱。战斗、行军、征兵、掠夺和增援相关特性通过核心事件管线分发，增援武将默认只影响自己的增援队伍。
- 世界地图第一版使用 `100 x 100` 权威坐标网格，每个玩家城池占一个格子；前端草地也必须按同一坐标格渲染，一块草地就是距离 1，城池只能覆盖自己的格子；玩家端地图只保留世界地图主视图，PVP 攻击、掠夺和增援统一按世界地图曼哈顿距离与最低兵种速度计算行军时间，速度 1 为一格 5 分钟且最终时间封顶 3 小时。
- 增援行军支持正式城金加速入口，每批最多加速 2 次、每次消耗 10 城金并写入 `reinforcement_accelerate` 流水；客户端传入的 `speedMultiplier` 会被服务端忽略，召回/遣返会按实际已行进或实际去程时间计算返程。
- 城池 -> 军事建筑 -> 军事 分组展示攻城武器营和特殊建筑营，两者只能消耗账户金币升级，提供攻城/特殊兵种征兵速度提升和征兵消耗减免。
- 左侧主菜单预留“联盟”入口，当前展示占位页，后续接入联盟系统。

示例：

```bash
export HERO3_DATABASE_DSN='hero3_user:hero3_password@tcp(127.0.0.1:3306)/test_hero3?parseTime=true&charset=utf8mb4&loc=UTC'
go run ./cmd/server
```

默认后端地址：

```text
http://localhost:8080
```

在线接口文档：

```text
http://localhost:8080/docs
```

后端启动后会通过 Scalar 展示 `docs/接口文档/openapi打包.yaml`，可在浏览器内查看接口并直接调试请求。

玩家端帮助文档：

```text
http://localhost:5173/help
```

帮助页入口位于玩家端侧栏顶部快捷入口。内容来自 `helpdocs/content/*.md`，可以直接手动新增、修改和删除 Markdown 文件；后端通过 `/api/v1/help/docs` 提供文档列表和正文读取。

当前旧版帮助正文已归档到 `过时文档/helpdocs/content/`，`helpdocs/content/` 仅保留空目录占位，后续应按当前实现重新编写玩家端帮助内容。

基础接口：

- `GET /healthz`：健康检查
- `GET /api/v1/meta`：服务元信息
- `GET /api/v1/game/bootstrap`：游戏模块启动信息
- `GET /api/v1/game/state`：玩家主界面游戏状态快照
- `POST /api/v1/accounts/register`：注册轻账号
- `POST /api/v1/accounts/login`：登录轻账号
- `GET /api/v1/accounts/{accountId}/players`：查看账号绑定的游戏存档
- `POST /api/v1/players/create`：创建账号绑定的游戏存档

## GM 后台

后台目录：

```bash
cd admin
```

常用命令：

```bash
pnpm install
pnpm dev
pnpm build
pnpm lint
```

默认后台地址：

```text
http://localhost:5174
```

## 参考项目

后续玩法设计会参考 `/Users/xieyuqiyu/Documents/Game/webgame_wlsg` 中已有的资源、城池、军事、地图、战斗、存档与通知等模块，但 Hero3 会按新的前后端分离结构重新实现。

## 设计文档

- [文档目录](./docs/文档目录.md)：按产品、系统、运维、流程、素材和接口文档分类。
- [项目最终目的](./docs/产品/项目最终目的.md)：记录长期项目目标、核心边界、模块化方向和生产级要求。
- [未来开发规划](./docs/产品/未来开发规划.md)：记录后续功能点和玩法模块规划。
- [核心地基设计](./docs/架构/核心地基设计.md)：记录稳定核心、玩法模块接入、事务、奖励、事件和 Modifier 管线。
- [服务器部署文档](./docs/运维部署/服务器部署文档.md)：记录当前线上部署结构、发版流程、回滚和排查命令。
- [OpenAPI 入口文档](./docs/接口文档/openapi/openapi.yaml)：按模块拆分维护，用于接口调试、文档查看和前后端对齐。
- [OpenAPI 打包文档](./docs/接口文档/openapi打包.yaml)：由 `make openapi` 生成，导入 Apifox 使用。
- [帮助文档站内容](./helpdocs/README.md)：玩家端帮助页读取的 Wiki 内容源，当前正文待按现行实现重写。

## Apifox

当前接口文档按模块维护在 `docs/接口文档/openapi/`：

```text
docs/接口文档/openapi/
├── openapi.yaml      # 入口文件
├── paths/            # 按模块维护接口路径
└── schemas/          # 按领域维护请求/响应模型
```

每次新增或修改接口后，运行：

```bash
make openapi
```

它会校验拆分后的 OpenAPI 并生成 `docs/接口文档/openapi打包.yaml`。在 Apifox 中选择“导入数据”，格式选择 `OpenAPI/Swagger`，导入 `docs/接口文档/openapi打包.yaml` 即可。

Go 后端启动后也会挂载在线接口文档：

```text
http://localhost:8080/docs
```

页面使用 Scalar 读取 `/openapi.yaml`，实际内容来自打包后的 `docs/接口文档/openapi打包.yaml`。

本地调试环境：

```text
Base URL: http://localhost:8080
```
