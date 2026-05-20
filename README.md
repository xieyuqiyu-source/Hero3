# Hero3

`Hero3` 是一个三国题材策略网页游戏项目。当前项目采用前后端分离结构，前端负责页面展示与交互，后端负责数据接口、存档、战斗结算和后续核心玩法规则。

## 项目结构

```text
Hero3/
├── web/    # React + TypeScript + Vite 玩家前端
├── admin/  # React + TypeScript + Vite GM 后台
└── go/     # Go 后端 API 服务
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

默认后端地址：

```text
http://localhost:8080
```

基础接口：

- `GET /healthz`：健康检查
- `GET /api/v1/meta`：服务元信息
- `GET /api/v1/game/bootstrap`：游戏模块启动信息

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

- [MVP 设计文档](./docs/mvp-design.md)：记录第一版核心循环、数据归属、页面范围、接口边界、代码规模规范、移动端适配要求和开发顺序。
