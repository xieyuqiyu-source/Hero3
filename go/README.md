# Hero3 Go 后端

`Hero3 Go 后端` 是 Hero3 的数据与游戏规则服务。当前阶段只提供最小 API 骨架，具体账号、存档、资源、建筑、征兵、地图、战斗等业务逻辑会在后续开发中逐步接入。

## 技术选择

当前骨架优先使用 Go 标准库：

- `net/http`：HTTP 服务与路由
- `log/slog`：结构化日志
- 环境变量：基础配置

后续在业务需要明确后，再接入：

- PostgreSQL：持久化玩家、存档、战报、地图状态
- `pgx/sqlc`：类型安全 SQL 访问
- `golang-migrate`：数据库迁移
- Redis：在线状态、排行榜、短期缓存或队列

## 目录结构

```text
go/
├── cmd/server/             # 服务启动入口
├── internal/api/           # HTTP 路由与接口处理
├── internal/config/        # 配置读取
├── internal/game/          # 游戏业务服务占位
├── internal/httpserver/    # HTTP Server 与中间件
├── migrations/             # 数据库迁移占位
├── sql/                    # SQL 查询占位
├── .env.example            # 环境变量示例
├── go.mod
└── README.md
```

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
HERO3_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
```

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

## 开发约定

- 页面展示状态放在前端，核心游戏数据以后端为准。
- 战斗、资源结算、存档版本迁移等关键逻辑应放在 Go 后端。
- 新业务优先放入 `internal/<domain>`，HTTP 层只做参数解析和响应封装。
- 数据库结构确定前，`migrations/` 与 `sql/` 先保留为空目录。
