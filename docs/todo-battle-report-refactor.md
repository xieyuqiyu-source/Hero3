# 战报系统重构待办

最后更新：2026-05-24

## 背景

当前战报存在 `state_json` 的 `recentBattleReports` 数组里，需要改为独立数据库表存储。

## 待改动项

### 1. 数据库：新建 battle_reports 表

```sql
CREATE TABLE battle_reports (
    id VARCHAR(64) PRIMARY KEY,
    player_id VARCHAR(64) NOT NULL,
    report_json JSON NOT NULL,
    is_read TINYINT(1) NOT NULL DEFAULT 0,
    deleted_by_player TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL,
    INDEX idx_reports_player (player_id, created_at DESC)
);
```

- 玩家删除 = 标记 `deleted_by_player = 1`，数据不物理删除
- GM 可查所有（含已删除）

---

### 2. Repository 接口扩展

新增方法：
- `SaveReport(report BattleReport) error`
- `ListReports(playerID string, limit int) ([]BattleReport, error)` — 玩家视角，过滤已删除 + 3天内
- `ListAllReports(playerID string) ([]BattleReport, error)` — GM 视角，不过滤
- `MarkReportsRead(playerID string) error`
- `DeleteReport(playerID string, reportID string) error` — 标记删除
- `DeleteAllReports(playerID string) error` — 标记全部删除
- `CountUnreadReports(playerID string) (int, error)` — 用于未读计数

---

### 3. MySQLRepository 实现

- `SaveReport`：INSERT INTO battle_reports
- `ListReports`：SELECT WHERE player_id = ? AND deleted_by_player = 0 AND created_at > 3天前 ORDER BY created_at DESC LIMIT ?
- `ListAllReports`：SELECT WHERE player_id = ? ORDER BY created_at DESC
- `MarkReportsRead`：UPDATE SET is_read = 1 WHERE player_id = ? AND is_read = 0
- `DeleteReport`：UPDATE SET deleted_by_player = 1 WHERE id = ? AND player_id = ?
- `DeleteAllReports`：UPDATE SET deleted_by_player = 1 WHERE player_id = ?
- `CountUnreadReports`：SELECT COUNT(*) WHERE player_id = ? AND is_read = 0 AND deleted_by_player = 0

---

### 4. MemoryRepository 实现

- 用 `map[string][]BattleReport` 按 playerID 存储
- 同样支持标记删除和已读

---

### 5. Service 层改动

- `applyNpcBattleResult`：战报生成后调用 `repo.SaveReport(report)` 写入数据库
- `GetState`：不再从 state_json 读战报，改为调用 `repo.ListReports` 填充 `RecentBattleReports`
- `MarkReportsRead`：改为调用 `repo.MarkReportsRead`
- `DeleteReport`：改为调用 `repo.DeleteReport`
- `DeleteAllReports`：改为调用 `repo.DeleteAllReports`
- `GameState.RecentBattleReports`：保留字段，但数据来源从 state_json 改为数据库查询

---

### 6. GameState 清理

- 新存档不再初始化 `RecentBattleReports`
- 旧存档的 state_json 里如果有战报数据，迁移到数据库表（一次性）
- state_json 里的 `recentBattleReports` 字段后续可以去掉（或保留为空数组兼容）

---

### 7. GM 后台接口

- `GET /api/v1/admin/reports?playerId=xxx` — 查看指定玩家所有战报（含已删除）
- admin 前端加一个战报查看面板

---

### 8. 侦查生成战报

- `ScoutNpc` 成功后也生成一条 `type: "scout"` 的战报
- 记录消耗的侦察兵数量和获取的情报摘要

---

### 9. MigrateMySQL 更新

- 在 `MigrateMySQL` 函数中加入 `battle_reports` 表的 CREATE TABLE 语句

---

## 执行顺序

1. 数据库建表（MigrateMySQL）
2. Repository 接口 + MySQL 实现 + Memory 实现
3. Service 层改用 Repository 方法
4. 从 state_json 移除战报存储
5. GM 后台接口
6. 侦查生成战报
7. 旧数据迁移（可选）

---

## 不变的部分

- 前端 API 接口路径不变
- 前端 BattleReport 类型不变
- 前端 NewsPage / BattleReportDetail 组件不变
- 战报生成逻辑（字段内容）不变

---

## 10. 防守方信息 25% 阈值

当前问题：无论战损多少，战报都完整暴露防守方兵种信息。

改动：
- BattleReport 新增字段 `defenderRevealed: bool`
- 后端：战报生成时计算防守方总损失比例，< 25% 时 `defenderUnits` 和 `defenderLostUnits` 设为空 map，`defenderRevealed = false`
- 前端：`BattleReportDetail` 根据 `defenderRevealed` 判断，为 false 时显示"对方战损低于25%，无法显示对方详细兵力情报"
- 阈值 25% 应做成可配置（GM 可调）
