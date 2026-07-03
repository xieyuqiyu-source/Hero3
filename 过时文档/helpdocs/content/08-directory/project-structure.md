# 目录结构

项目根目录主要分为几个部分。

## 核心目录

- `go/`：Go 后端 API 服务。
- `web/`：玩家前端。
- `admin/`：GM 后台。
- `docs/`：设计文档。
- `helpdocs/`：玩家端帮助中心内容源。
- `scripts/`：脚本工具。
- `memory/`：项目记忆。

## Go 目录

- `cmd/`：程序入口和命令行工具。
- `internal/transport/`：HTTP API 输入输出层。
- `internal/app/`：应用服务和业务编排。
- `internal/core/`：核心领域能力。
- `internal/platform/`：认证、配置、HTTP server 等平台能力。
- `internal/infrastructure/`：MySQL 等基础设施实现。
- `config/`：游戏配置。

## Web 目录

- `src/pages/`：页面。
- `src/components/`：通用组件。
- `src/store/`：前端状态。
- `src/api/`：API 客户端。
- `src/types/`：类型定义。
