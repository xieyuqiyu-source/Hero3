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
- 配置 `HERO3_DATABASE_DSN` 后启用 MySQL/MariaDB，启动时会自动创建当前需要的账号和存档表。
- 本地开发模式应连接 `test_` 前缀测试库，例如 `test_hero3`，不要直接写稳定玩家库。
- `make migrate` 迁移当前 DSN 指向的库，`make migrate-test` 创建并迁移 `test_` 前缀测试库。
- `make clone-data` 可从 `HERO3_SOURCE_DATABASE_DSN` 复制数据到当前 `test_` 目标库，复制后自动回填并校验资源、背包、建筑、资源田格子、兵力、征兵队列、武将和 Buff 权威表。
- `make healthcheck-authority` 是当前权威表健康检查；`verify-*` 仍保留为旧快照迁移期校验，不作为轻量 `state_json` 后的日常通过标准。
- `clone-data` 允许目标测试库比源库多出迁移后的新列，复制时只写公共列；源库列在目标库不存在时会中止。
- `clone-data` 不复制或清空 `schema_migrations`，测试库迁移记录由测试库自己的迁移命令维护。
- 物品系统使用 `go/config/items.json` 和 `go/config/drop_pools.json` 配置注册；背包权威表按格子 `slot_id` 存储，兼容接口仍返回按物品聚合的 `inventory`。
- 物品获得和消耗会写入 `item_ledger`，GM 后台可查看物品配置、玩家背包格子和物品流水。
- 轮回绝境副本使用 `go/config/reincarnation.json` 配置层级、波次、加成、金币重置随机加成价格和奖励；玩家入口位于“地图 -> 副本”，GM 后台可编辑 JSON 配置、查看实例并处理异常结算。
- 军事招募页展示攻城武器营和特殊建筑营，两者只能消耗账户金币升级，提供攻城/特殊兵种征兵速度提升和征兵消耗减免。

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
- [帮助文档站内容](./helpdocs/README.md)：玩家端帮助页读取的 Wiki 内容源。

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
