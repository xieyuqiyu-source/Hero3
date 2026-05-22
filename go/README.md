# Hero3 Go 后端

`Hero3 Go 后端` 是 Hero3 的数据与游戏规则服务。当前阶段只提供最小 API 骨架，具体账号、存档、资源、建筑、征兵、地图、战斗等业务逻辑会在后续开发中逐步接入。

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
# HERO3_DATABASE_DSN=hero3_user:hero3_password@tcp(127.0.0.1:3306)/hero3?parseTime=true&charset=utf8mb4&loc=UTC
```

## 数据库

不配置 `HERO3_DATABASE_DSN` 时，服务使用内存存储，重启后账号和新建存档会丢失。

配置 `HERO3_DATABASE_DSN` 后，服务使用 MySQL/MariaDB，并在启动时自动创建当前需要的表：

- `accounts`：轻账号
- `players`：账号绑定的游戏存档，当前阶段用 `state_json` 保存完整游戏状态

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

日常开发只需要：

```bash
./dev.sh
```

如果后续关闭公网 `3306`，可以把 `go/.env` 的 `HERO3_DB_TUNNEL_ENABLED` 改为 `true`，让 `dev.sh` 自动启动 SSH 隧道。

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

## 开发约定

- 页面展示状态放在前端，核心游戏数据以后端为准。
- 战斗、资源结算、存档版本迁移等关键逻辑应放在 Go 后端。
- 新业务优先放入 `internal/<domain>`，HTTP 层只做参数解析和响应封装。
- 数据库结构确定前，`migrations/` 与 `sql/` 先保留为空目录。
