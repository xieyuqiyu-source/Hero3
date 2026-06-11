# Hero3 开发进度记录

本文档用于记录 Hero3 每次提交的开发进度。后续凡是执行提交前，都需要先把本次改动追加到本文档中，再随代码一起提交。

记录要求：

- 每次提交单独一个章节，章节之间用分隔线隔开。
- 写清楚提交号、提交时间、改动范围、业务目标、具体改动、验证结果和注意事项。
- 如果本次改动影响后端、Web、Admin、OpenAPI、部署流程，需要分别说明。
- 不只写 Git commit message，要写清楚这次改动对游戏功能和后续开发的影响。

---

## 2026-06-11 - `feat: add mail system ui foundation`

### 改动目标

先完成信函系统的开发设计文档和玩家端第一版 UI，为后续后端逻辑开发提供稳定参照。该版本只做 UI 和文档，不接入真实信函接口，不改变后端业务逻辑。

### 文档改动

- 新增 `docs/mail-system-design.md`。
- 明确信函、军情、公告三个模块的边界：
  - 军情负责战斗和侦查结果。
  - 信函负责玩家收件箱、GM 通知、补偿、奖励、活动奖励、玩家互发。
  - 公告负责全服或指定范围公开通知。
- 设计了信函系统第一阶段 MVP：
  - 玩家分页查看信函。
  - 玩家查看详情并标记已读。
  - 玩家删除单封信函。
  - GM 给指定玩家发信。
  - 预留附件、过期时间、来源追踪。
- 设计了后续扩展方向：
  - 附件领取。
  - 全服邮件。
  - 按阵营/等级/活跃范围批量发信。
  - 玩家互发。
  - 邮件模板。
  - 黑名单、举报和风控。
- 记录了附件领取的事务和幂等要求，避免后续出现金币、资源、兵种重复领取问题。
- 新增 `docs/development-progress.md`，后续每次提交前都需要追加本次开发进度。

### Web UI 改动

- 新增玩家端信函页面：
  - `web/src/pages/mail/MailPage.tsx`
  - `web/src/pages/mail/components/MailDetail.tsx`
  - `web/src/pages/mail/data.ts`
  - `web/src/pages/mail/index.ts`
- 新增 `/mail` 路由。
- 侧边栏和移动菜单的“信函”入口可以跳转到 `/mail`。
- 信函列表使用模拟数据展示，暂不接后端接口。
- 支持信函分类筛选：
  - 全部
  - GM 通知
  - 补偿
  - 奖励
  - 活动
  - 私信
- 不同信函类型使用不同颜色区分：
  - GM 通知：蓝色
  - 补偿：琥珀色
  - 奖励：绿色
  - 活动：紫色
  - 私信：玫红色
- 信函详情改为弹窗展示：
  - 桌面端居中弹窗。
  - 手机端底部弹出，接近全屏高度。
  - 点击遮罩或右上角关闭按钮可以关闭。
- 详情弹窗展示：
  - 标题
  - 发件人
  - 时间
  - 类型标签
  - 正文
  - 附件预留区
  - 删除按钮
  - 领取附件按钮预留
- 增加信函 UI 动画：
  - 列表项逐条上滑淡入。
  - 列表项 hover 轻微上浮并加阴影。
  - 弹窗遮罩淡入并模糊背景。
  - 弹窗上滑、缩放、淡入。
- 新增全局动画 class：
  - `animate-mail-backdrop-in`
  - `animate-mail-dialog-in`

### 资源改动

- 当前工作区存在甄宓头像资源：
  - `web/src/assets/generals/wei/zhenmi.png`
  - `web/src/assets/generals/wei/zhenmi.webp`
- 这两个资源已经被现有将领配置引用，构建时会进入产物。本次一起纳入版本，避免其他环境缺少甄宓头像资源。

### 验证结果

- `web npm run build` 通过。
- 本次没有修改 Go 后端业务逻辑，因此没有新增后端测试。

### 后续注意事项

- 下一步开发信函系统后端第一阶段：
  - 建立 `mails` 表。
  - 增加 `Mail`、`MailPage`、`MailAttachment` 类型。
  - 增加玩家分页列表、详情已读、删除接口。
  - 增加 GM 给指定玩家发信接口。
- 后端接口完成后，再把当前模拟数据替换为真实 API。
- 信函红点暂时不接军情未读数，后续应使用独立的 `unreadMailCount` 或统一通知计数。

---

## 2026-06-11 - `baef545 feat: paginate battle reports`

### 改动目标

给军情战报系统增加服务端分页，避免玩家战报数量增长后，游戏状态接口和军情页面一次性拉取过多战报，造成服务器和前端渲染压力。同时把 GitHub Actions 部署流程改为手动触发，避免每次推送 `main` 都自动部署线上环境。

### 后端改动

- 新增军情分页接口：
  - `GET /api/v1/news/reports`
  - 查询参数：`playerId`、`page`、`pageSize`
  - 默认 `page=1`
  - 默认 `pageSize=10`
  - 最大 `pageSize=50`
- 新增分页响应模型 `BattleReportPage`：
  - `reports`
  - `page`
  - `pageSize`
  - `total`
- 调整战报仓储接口：
  - `ListReports(playerID, limit)` 改为 `ListReports(playerID, limit, offset)`
  - 返回值增加 `total`
- MySQL 战报查询改为数据库分页：
  - 使用 `LIMIT ? OFFSET ?`
  - 额外执行 `COUNT(*)` 获取总条数
  - 只统计玩家未删除、最近 3 天内的战报
- 内存仓储同步实现分页逻辑，保证测试环境和 MySQL 行为一致。
- `GameState.recentBattleReports` 不再装载 50 条战报，改为只装载最近 10 条，用于兼容旧页面和基础提示。
- 新增统一的战报摘要装载逻辑：
  - 状态响应只带少量最近战报
  - `unreadMessageCount` 使用仓储真实统计结果
- 未读军情数量统计口径和军情列表保持一致：
  - 只统计最近 3 天内
  - 未删除
  - 未读

### Web 改动

- 军情页不再直接使用 `GameState.recentBattleReports` 作为完整列表。
- 新增 `gameApi.listReports(playerId, page, pageSize)`。
- 军情页改为按页请求后端分页接口。
- 每页显示 10 条战报。
- 页面底部增加上一页、下一页、当前页、总页数、总条数显示。
- 删除单条战报后刷新当前页。
- 清空全部战报后回到第一页并刷新列表。
- 请求军情战报时增加 loading：
  - 首次进入显示“军情加载中...”
  - 翻页或刷新时显示列表内轻遮罩和旋转图标
  - loading 期间禁用翻页、清空和战报点击，避免重复请求或重复操作
- 导航栏和侧边栏的军情红点改为使用 `unreadMessageCount`，不再只检查前端已有的最近战报列表。

### OpenAPI 文档改动

- 新增 `News` 分组。
- 新增 `docs/openapi/paths/news.yaml`。
- 在主 OpenAPI 中挂载：
  - `/api/v1/news/reports`
- 新增 `BattleReportPage` schema。

### 部署流程改动

- `.github/workflows/deploy.yml` 移除 `push main` 自动触发。
- 保留 `workflow_dispatch`。
- 后续推送代码不会自动部署，需要在 GitHub Actions 页面手动运行部署。

### 验证结果

- `go test ./...` 通过。
- `web npm run build` 通过。
- 推送后检查 GitHub Actions 最近运行记录，未发现本次提交触发新的自动部署，说明手动触发配置生效。

### 后续注意事项

- 目前军情列表仍然只展示最近 3 天战报，这是原有业务口径。后续如果要做长期战报归档，需要增加时间范围筛选。
- 删除和标记已读接口目前仍返回 `GameState`，为了兼容现有前端暂时保留。后续如果继续优化服务器压力，可以把这些接口改成只返回操作结果和最新未读数。
- `BattleReport` 的 OpenAPI schema 仍然偏简化，后续战报详情字段稳定后可以补完整。
