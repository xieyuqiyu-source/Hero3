# Hero3 信函系统开发设计文档

最后更新：2026-06-11

## 1. 模块定位

信函系统是 Hero3 的玩家消息收件箱，承担系统通知、GM 通知、补偿、奖励、活动奖励、玩家互发等消息投递能力。

信函系统必须与军情、公告保持边界清晰：

| 模块 | 作用 | 示例 |
|------|------|------|
| 军情 | 战斗和侦查自动生成的结果记录 | NPC 战报、侦查战报、PVP 战报、将领特性触发 |
| 信函 | 投递给某个玩家或某批玩家的收件箱消息 | GM 通知、维护补偿、活动奖励、玩家私信 |
| 公告 | 面向全服或指定范围的公开通知 | 停服公告、版本更新、活动开启说明 |

信函系统不是聊天系统。第一阶段以“异步消息 + 奖励承载 + GM 管理”为核心，不做实时聊天。

## 2. 设计目标

### 2.1 第一阶段目标

第一阶段做一个可用、可扩展、不会干扰现有玩法的信函 MVP：

- 玩家可以分页查看信函。
- 玩家可以打开信函详情。
- 打开信函后自动标记已读。
- 玩家可以删除单封信函。
- GM 可以给指定玩家发送信函。
- 信函支持类型字段，用于区分补偿、奖励、GM 通知、活动奖励、玩家互发等来源。
- 附件、过期时间、来源追踪字段先做结构预留。

### 2.2 后续扩展目标

后续信函系统需要自然扩展到：

- 附件领取。
- 全服邮件。
- 按阵营、等级、活跃时间、账号范围批量发信。
- 玩家互发信函。
- 邮件模板。
- 邮件过期。
- 黑名单和风控。
- 邮件操作审计。

## 3. 核心原则

1. **信函独立存储**
   不放进 `GameState` 主体中，避免状态接口越来越大。玩家端通过独立分页接口读取。

2. **分页优先**
   信函列表必须服务端分页。默认每页 10 条，最大每页 50 条。

3. **附件领取必须幂等**
   后续做附件领取时，同一封信函只能成功领取一次。即使玩家重复点击、网络重试、并发请求，也不能重复发奖励。

4. **删除是玩家侧删除**
   玩家删除信函不一定物理删除数据库记录，建议使用 `deleted_by_player` 软删除，便于审计和排查。

5. **系统邮件与玩家邮件共用基础表**
   通过 `sender_type`、`mail_type`、`source_type` 区分来源，不为每种邮件单独建表。

6. **先做文本，富文本后置**
   第一阶段正文使用纯文本。富文本、图片、链接、颜色样式后续再做，避免 GM 后台和前端展示复杂化。

7. **玩家互发受限制**
   玩家互发不能无上限开放。必须预留发送冷却、每日上限、内容长度、封禁/屏蔽等机制。

## 4. 信函类型设计

### 4.1 mailType

`mailType` 表示信函业务类型：

| 类型 | 含义 | 第一阶段是否实现 |
|------|------|------------------|
| `gm_notice` | GM 通知 | 是 |
| `compensation` | 补偿邮件 | 预留，第一阶段可由 GM 手动发送 |
| `reward` | 普通奖励邮件 | 预留 |
| `event_reward` | 活动奖励邮件 | 预留 |
| `system_notice` | 系统通知 | 预留 |
| `player_message` | 玩家互发 | 预留，第二阶段做 |

### 4.2 senderType

`senderType` 表示发件方身份：

| 类型 | 含义 |
|------|------|
| `system` | 系统自动发送 |
| `gm` | GM 后台发送 |
| `player` | 玩家发送 |

### 4.3 sourceType

`sourceType` 用于追踪邮件从哪里产生：

| 类型 | 示例 |
|------|------|
| `manual` | GM 手动发信 |
| `maintenance` | 维护补偿 |
| `activity` | 活动奖励 |
| `combat` | 战斗相关通知 |
| `system` | 系统流程通知 |
| `player` | 玩家互发 |

`sourceId` 可记录活动 ID、GM 操作 ID、战斗 ID、玩家 ID 等。

## 5. 数据模型

### 5.1 Mail

建议 Go 结构：

```go
type Mail struct {
    ID              string         `json:"id"`
    PlayerID        string         `json:"playerId"`
    MailType        string         `json:"mailType"`
    SenderType      string         `json:"senderType"`
    SenderID        string         `json:"senderId,omitempty"`
    SenderName      string         `json:"senderName"`
    Title           string         `json:"title"`
    Content         string         `json:"content"`
    Attachments     []MailAttachment `json:"attachments,omitempty"`
    SourceType      string         `json:"sourceType,omitempty"`
    SourceID        string         `json:"sourceId,omitempty"`
    IsRead          bool           `json:"isRead"`
    IsClaimed       bool           `json:"isClaimed"`
    DeletedByPlayer bool           `json:"deletedByPlayer,omitempty"`
    ExpiresAt       string         `json:"expiresAt,omitempty"`
    CreatedAt       string         `json:"createdAt"`
    ReadAt          string         `json:"readAt,omitempty"`
    ClaimedAt       string         `json:"claimedAt,omitempty"`
}
```

第一阶段可以先不开放附件领取，但 `Attachments`、`IsClaimed`、`ClaimedAt` 字段建议提前定好。

### 5.2 MailAttachment

附件建议用结构化 JSON 保存：

```go
type MailAttachment struct {
    Type     string         `json:"type"`
    ItemID   string         `json:"itemId"`
    Amount   int            `json:"amount"`
    Metadata map[string]any `json:"metadata,omitempty"`
}
```

附件类型：

| Type | 含义 | ItemID 示例 |
|------|------|-------------|
| `resource` | 资源 | `wood`、`stone`、`iron`、`food` |
| `city_gold` | 城金 | `city_gold` |
| `gold` | 账号金币 | `gold` |
| `unit` | 兵种 | `qingZhouArmy` |
| `general_exp` | 将领经验 | `general_exp` |
| `buff` | 限时加成 | `production_boost` |
| `item` | 道具 | 后续背包系统 ID |

### 5.3 MySQL 表建议

```sql
CREATE TABLE IF NOT EXISTS mails (
  id VARCHAR(64) PRIMARY KEY,
  player_id VARCHAR(64) NOT NULL,
  mail_type VARCHAR(32) NOT NULL,
  sender_type VARCHAR(32) NOT NULL,
  sender_id VARCHAR(64) NOT NULL DEFAULT '',
  sender_name VARCHAR(64) NOT NULL DEFAULT '',
  title VARCHAR(120) NOT NULL,
  content TEXT NOT NULL,
  attachments_json JSON NULL,
  source_type VARCHAR(32) NOT NULL DEFAULT '',
  source_id VARCHAR(64) NOT NULL DEFAULT '',
  is_read BOOLEAN NOT NULL DEFAULT FALSE,
  is_claimed BOOLEAN NOT NULL DEFAULT FALSE,
  deleted_by_player BOOLEAN NOT NULL DEFAULT FALSE,
  expires_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  read_at DATETIME NULL,
  claimed_at DATETIME NULL,
  INDEX idx_mails_player_list (player_id, deleted_by_player, created_at DESC),
  INDEX idx_mails_player_unread (player_id, deleted_by_player, is_read),
  INDEX idx_mails_source (source_type, source_id)
);
```

后续如果做全服批量邮件，可以再拆出：

- `mail_batches`
- `mail_recipients`

第一阶段不建议一开始就做批量表，避免复杂化。

## 6. 后端接口设计

### 6.1 玩家端接口

#### 分页获取信函

```text
GET /api/v1/mails?playerId={playerId}&page=1&pageSize=10
```

返回：

```json
{
  "mails": [],
  "page": 1,
  "pageSize": 10,
  "total": 0,
  "unread": 0
}
```

列表项可以返回完整 `Mail`，也可以返回摘要 `MailSummary`。第一阶段为了简单可直接返回完整 `Mail`，但前端列表只展示摘要字段。

#### 获取信函详情

```text
GET /api/v1/mails/{mailId}?playerId={playerId}
```

说明：

- 必须校验该信函属于当前玩家。
- 如果未读，可以在读取详情时自动标记已读。

#### 标记已读

```text
POST /api/v1/mails/{mailId}/read
```

请求体：

```json
{ "playerId": "player_xxx" }
```

#### 删除单封信函

```text
POST /api/v1/mails/{mailId}/delete
```

请求体：

```json
{ "playerId": "player_xxx" }
```

#### 领取附件（第二阶段）

```text
POST /api/v1/mails/{mailId}/claim
```

请求体：

```json
{ "playerId": "player_xxx" }
```

要求：

- 必须事务处理。
- 检查未领取、未过期、未删除。
- 发放附件。
- 标记 `is_claimed = true`。
- 返回更新后的奖励结果和玩家局部状态。

### 6.2 Admin 接口

#### GM 给指定玩家发信

```text
POST /api/v1/admin/mails/send
```

请求体：

```json
{
  "playerId": "player_xxx",
  "mailType": "gm_notice",
  "title": "GM 通知",
  "content": "这里是正文",
  "attachments": [],
  "expiresAt": ""
}
```

#### GM 查看玩家信函

```text
GET /api/v1/admin/players/{playerId}/mails?page=1&pageSize=10
```

#### 后续批量发信

```text
POST /api/v1/admin/mails/send-batch
```

第一阶段不做批量发信，但接口方向提前保留。

## 7. Repository 设计

建议新增 MailRepository 相关方法，也可以先放入现有 Repository interface。

```go
type MailPage struct {
    Mails    []Mail `json:"mails"`
    Page     int    `json:"page"`
    PageSize int    `json:"pageSize"`
    Total    int    `json:"total"`
    Unread   int    `json:"unread"`
}

type Repository interface {
    SaveMail(mail Mail) error
    GetMailByID(mailID string) (Mail, error)
    ListMails(playerID string, limit int, offset int) ([]Mail, int, error)
    CountUnreadMails(playerID string) (int, error)
    MarkMailRead(playerID string, mailID string, readAt time.Time) error
    DeleteMail(playerID string, mailID string) error
    ClaimMailAttachments(playerID string, mailID string, apply func(Mail) error) error
}
```

第一阶段不实现 `ClaimMailAttachments` 也可以，但接口设计时要考虑未来事务边界。

第一阶段删除接口只返回删除状态，删除后停留第几页由前端自己重新拉取，避免 Service 层把分页固定成第 1 页。

## 8. Service 设计

建议新增独立文件：

- `go/internal/game/model_mail.go`
- `go/internal/game/service_mail.go`

主要职责：

- 校验玩家 ID。
- 校验邮件标题和正文长度。
- 生成邮件 ID。
- 处理分页参数。
- 统一设置发件人、创建时间、过期时间。
- 调用仓储读写。
- 后续处理附件领取事务。

标题和正文限制建议：

| 字段 | 限制 |
|------|------|
| title | 1-60 个中文字符左右，数据库限制 120 字符 |
| content | 1-5000 字符 |
| attachments | 第一阶段最多 10 个 |

## 9. Web 玩家端设计

### 9.1 页面入口

现有导航里已经有“信函”入口。第一阶段将其接入真实页面。

建议路径：

```text
/mail
```

文件建议：

```text
web/src/pages/mail/MailPage.tsx
web/src/pages/mail/components/MailListItem.tsx
web/src/pages/mail/components/MailDetail.tsx
```

### 9.2 列表展示

列表每条展示：

- 邮件类型标签。
- 标题。
- 发件人。
- 创建时间。
- 是否未读。
- 是否有附件。
- 是否已领取。
- 是否即将过期。

### 9.3 详情展示

详情展示：

- 标题。
- 发件人。
- 创建时间。
- 正文。
- 附件区域。
- 删除按钮。
- 返回按钮。

第一阶段附件区只展示“附件功能开发中”或隐藏。

### 9.4 Loading 和空状态

必须有：

- 首次加载 loading。
- 翻页 loading。
- 空状态：暂无信函。
- 错误状态：信函加载失败，可重试。

## 10. Admin 设计

Admin 第一阶段需要一个“信函管理”页面：

```text
admin/src/pages/mails/MailAdminPage.tsx
```

功能：

- 输入玩家 ID。
- 填写标题。
- 填写正文。
- 选择信函类型。
- 选择过期时间，可选。
- 发送。
- 查看指定玩家信函列表。

第一阶段可以不做全服发送，避免误发。

GM 后台所有发送操作必须保留二次确认，即使玩家端有“不再提醒”偏好，GM 高危操作也不共享该偏好。

## 11. 玩家互发设计

玩家互发属于第二阶段，不建议第一阶段直接开放。

原因：

- 需要防骚扰。
- 需要内容限制。
- 需要举报/屏蔽。
- 可能涉及敏感内容。
- 需要发送频率限制。

第二阶段建议规则：

| 规则 | 建议 |
|------|------|
| 每日发送上限 | 20 封 |
| 单玩家冷却 | 30 秒 |
| 标题长度 | 1-30 字 |
| 正文长度 | 1-500 字 |
| 是否允许附件 | 第一版玩家互发不允许附件 |
| 黑名单 | 支持屏蔽某玩家 |
| 举报 | 后续接入 |

玩家互发接口：

```text
POST /api/v1/mails/send-player
```

请求体：

```json
{
  "fromPlayerId": "player_a",
  "toPlayerId": "player_b",
  "title": "你好",
  "content": "正文"
}
```

## 12. 附件领取设计

附件领取是信函系统最容易出问题的地方，需要第二阶段单独做严谨。

### 12.1 领取流程

```text
玩家点击领取
→ 后端开启事务
→ 查询邮件并加锁
→ 校验玩家归属
→ 校验未删除
→ 校验未过期
→ 校验未领取
→ 发放附件
→ 标记已领取
→ 写入货币/资源流水
→ 提交事务
→ 返回结果
```

### 12.2 不同附件的处理

| 附件 | 发放到哪里 |
|------|------------|
| resource | 当前存档资源，受容量限制或转城金，规则后续确定 |
| city_gold | 当前存档城金 |
| gold | 账号金币 |
| unit | 当前存档军队 |
| general_exp | 当前存档将领 |
| buff | 当前存档加成列表 |
| item | 背包系统，等背包上线 |

### 12.3 流水记录

涉及金币、城金、资源、兵种的附件领取，后续都应该记录来源：

- `refType = mail`
- `refID = mailID`
- `reason = mail_claim`

## 13. 过期策略

建议规则：

| 邮件类型 | 默认过期 |
|----------|----------|
| gm_notice | 不过期 |
| compensation | 30 天 |
| reward | 30 天 |
| event_reward | 活动结束后 7 天 |
| system_notice | 7 天 |
| player_message | 30 天 |

第一阶段可以只存 `expiresAt`，不做自动清理。

后续可以增加定时任务或后台清理脚本：

- 删除 90 天前已删除邮件。
- 删除 180 天前无附件且已读邮件。
- 保留未领取奖励邮件更久，避免玩家投诉。

## 14. 红点与未读数

玩家端导航的“信函”红点应该使用后端未读数。

建议在 `GameState` 中增加：

```go
UnreadMailCount int `json:"unreadMailCount"`
```

或者未来统一成：

```go
NotificationCounts struct {
    News int `json:"news"`
    Mail int `json:"mail"`
    Notice int `json:"notice"`
}
```

第一阶段可以先加 `unreadMailCount`，后续再合并为统一通知计数。

## 15. OpenAPI 要求

开发信函接口时必须同步维护：

```text
docs/openapi/paths/mail.yaml
docs/openapi/schemas/mail.yaml
docs/openapi/openapi.yaml
```

至少包含：

- 玩家分页获取信函。
- 玩家获取详情。
- 玩家标记已读。
- 玩家删除信函。
- GM 发送信函。
- GM 查看玩家信函。

## 16. 第一阶段开发拆分

建议按以下顺序开发，避免跑偏：

### Step 1：后端数据模型和仓储

- 新增 `Mail`、`MailAttachment`、`MailPage`。
- MySQL 建表。
- MemoryRepository 实现。
- Repository interface 增加邮件方法。

### Step 2：后端 Service 和 API

- 玩家分页查询。
- 玩家详情读取并标记已读。
- 玩家删除。
- GM 给指定玩家发信。
- GM 查看指定玩家信函列表。

### Step 3：Web 玩家端

- 新增信函页。
- 接入现有导航“信函”入口。
- 列表分页。
- 详情页。
- 已读、删除。
- Loading、空状态、错误状态。

### Step 4：Admin GM 后台

- 新增信函管理页。
- 给指定玩家发送信函。
- 查看指定玩家信函。
- 保留二次确认。

### Step 5：OpenAPI 和开发进度文档

- 更新 OpenAPI。
- 更新 `docs/development-progress.md`。
- 运行 `go test ./...`。
- 运行 `web npm run build`。
- 运行 `admin npm run build`。

## 17. 第一阶段不做的内容

为了避免 MVP 变复杂，第一阶段暂不做：

- 附件领取。
- 全服邮件。
- 按阵营/等级/活跃玩家批量发信。
- 玩家互发。
- 黑名单。
- 举报。
- 富文本。
- 邮件模板。
- 自动过期清理。

这些都要预留设计，但不进入第一版开发范围。

## 18. 验收标准

第一阶段完成后，应满足：

- 玩家能打开信函页面并看到分页列表。
- 玩家点击信函后能看到详情，并且未读变已读。
- 玩家能删除单封信函。
- GM 能给指定玩家发送一封信函。
- 玩家刷新页面后仍能看到该信函。
- 信函列表不会进入 `GameState` 大对象。
- 未读数能在导航红点正确体现。
- OpenAPI 文档能导入 Apifox。
- 后端测试通过。
- Web 和 Admin 构建通过。

## 19. 风险点

| 风险 | 后果 | 处理方式 |
|------|------|----------|
| 附件重复领取 | 资源/金币被刷 | 第二阶段必须事务和幂等 |
| 全服邮件误发 | 大量玩家收到错误奖励 | 第一阶段不做全服，后续必须二次确认 |
| 玩家互发刷屏 | 骚扰和存储压力 | 第二阶段必须限频 |
| 信函塞进 GameState | 状态接口变慢 | 必须独立分页接口 |
| 删除物理删除 | 无法审计 | 使用软删除 |
| OpenAPI 不同步 | 前端和接口认知不一致 | 每次接口变更必须同步文档 |

## 20. 推荐第一版结论

第一版只做“系统/GM 信函 MVP”：

- 后端独立信函表。
- 玩家分页查看。
- 玩家详情已读。
- 玩家删除。
- GM 给指定玩家发信。
- 字段预留附件、过期、来源追踪。

等这一版稳定后，再做附件领取、批量邮件、玩家互发。
