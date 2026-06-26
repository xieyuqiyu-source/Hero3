# API 总览

后端 API 统一挂在 `/api/v1` 下。

## 主要 API 模块

- 账号接口。
- 游戏状态接口。
- 城池接口。
- 军事接口。
- 地图接口。
- 战斗接口。
- 军情接口。
- 信函接口。
- 金币接口。
- 道具接口。
- 小游戏接口。
- 帮助文档接口。
- GM 后台接口。

## 在线接口文档

后端启动后可以打开：

```text
http://localhost:8080/docs
```

这个页面展示 OpenAPI 文档。

## 帮助文档 API

帮助站使用：

```text
GET /api/v1/help/docs
GET /api/v1/help/docs/{documentId}
```

这些接口读取 `helpdocs/content` 里的 Markdown 文件。
