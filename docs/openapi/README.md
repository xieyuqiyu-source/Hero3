# Hero3 OpenAPI 维护规范

Hero3 接口会持续增加，OpenAPI 按模块拆分维护。

## 文件职责

```text
docs/openapi/
├── openapi.yaml          # 入口文件：服务信息、tags、paths/components 引用
├── paths/                # 接口路径，按业务模块拆分
│   ├── system.yaml
│   ├── account.yaml
│   ├── game.yaml
│   └── admin.yaml
└── schemas/              # 请求/响应模型，按领域拆分
    ├── system.yaml
    ├── account.yaml
    ├── game-state.yaml
    └── common.yaml
```

## 新增接口流程

1. 在 `paths/` 对应模块文件中新增接口。
2. 如果有新的请求或响应结构，在 `schemas/` 对应领域文件中新增 schema。
3. 在 `openapi.yaml` 的 `paths` 或 `components.schemas` 中补引用。
4. 运行：

```bash
make openapi
```

5. 将生成的 `docs/openapi.bundle.yaml` 导入 Apifox。

## 约定

- `docs/openapi/openapi.yaml` 是维护入口。
- `docs/openapi.bundle.yaml` 是 Apifox 导入文件，由脚本生成，不手动编辑。
- 后端新增接口前，先补 OpenAPI；接口路径、参数、响应状态码必须和 Go handler 对齐。
- admin 的接口诊断面板后续应从 OpenAPI 生成，避免维护两份接口清单。
