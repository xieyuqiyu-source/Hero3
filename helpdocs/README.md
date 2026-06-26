# Hero3 帮助文档站内容目录

这里存放玩家端“帮助”页面读取的 Markdown 文档，前端会按目录生成 Wiki 菜单树。

- `content/`：帮助站正文，支持手动新增、修改、删除 `.md` 文件。
- 一级目录会显示为左侧菜单分类，例如 `02-core/` 对应“核心系统”。
- 文件路径会作为文档 ID，例如 `02-core/core-boundary.md` 对应 `02-core/core-boundary`。
- 文档第一行 `# 标题` 会作为帮助站标题。

当前帮助站由 Go 后端读取本目录，并由 Web 前端渲染为 Wiki 风格页面。

## 当前分类

- `01-project/`：项目总览。
- `02-core/`：核心系统。
- `03-registries/`：注册表。
- `04-module-development/`：模块开发。
- `05-gameplay-planning/`：玩法规划。
- `06-database/`：数据库。
- `07-api/`：API 接口。
- `08-directory/`：目录结构。
- `09-development-rules/`：开发规则。
- `10-memory/`：项目记忆。
