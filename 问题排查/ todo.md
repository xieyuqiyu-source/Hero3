## 待解决

高危：加减城金是读-改-写，没有事务、锁、幂等。 AddGold/DeductGold 先 GetState，再改 state.CityGold，再 SaveState，并发下会丢更新或重复扣减。如果后面接充值、礼包、任务奖励，这里会出账务问题。上线前/PvP 前必须处理。

中等：OpenAPI 和前端/后台没有同步到这版货币能力。 文档里缺 /api/v1/gold/exchange、/api/v1/gold/reverse-exchange，GameState schema 缺 cityGold/lastExchangeAt，AccountSummary 缺 gold。后面会出现"代码有了，文档没有"的偏差。

中低：MySQL 迁移把 ALTER TABLE 的错误全吞了。 go/internal/storage/mysql.go 增量迁移无条件忽略错误，应该只忽略"列已存在"错误，其它错误要返回。

## 已解决

- ~~高危：/api/v1/gold/add 公开可调用~~ → 已移到 /admin/gold/add
- ~~中高：货币模型不完整~~ → 双向兑换接口已实现，GetAccountByID/UpdateAccountGold 已加
- ~~低：gofmt~~ → 已格式化
