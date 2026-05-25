高危：/api/v1/gold/add 现在是公开可调用的。 任何知道 playerId 的客户端都能直接加城金，这在业务上等于“客户端可造币”。go/internal/api/router.go:48-49、go/internal/api/handlers.go:612-663。这类接口应当只给 GM/后台，或者至少走明确的权限层，不应和普通游戏请求混在一起。

高危：加减城金是读-改-写，没有事务、锁、幂等。 AddGold/DeductGold 先 GetState，再改 state.CityGold，再 SaveState，并发下会丢更新或重复扣减。go/internal/game/service_gold.go:25-84、go/internal/game/repository.go:17-18、go/internal/storage/mysql.go:302-346。如果后面接充值、礼包、任务奖励，这里会出账务问题。

中高：货币模型拆得不完整。 你已经加了 Account.Gold，但没有任何账户级金币的增减、查询、兑换接口；而城金又放在 GameState.state_json 里，导致它天然不可单独查询、审计、对账。go/internal/game/model.go:8-13,102-107、go/internal/storage/mysql.go:87-107,302-346。现在看像是“两套货币字段”，但业务闭环其实还没成立。

中等：OpenAPI 和前端/后台没有同步到这版货币能力。 现在文档里没有 /api/v1/gold/add、/api/v1/gold/deduct，GameState schema 也没有 cityGold，AccountSummary 也没有 gold。docs/openapi/openapi.yaml:26-75、docs/openapi/schemas/game-state.yaml:1-55、docs/openapi/schemas/account.yaml:49-69。后面 Apifox、admin、web 一定会出现“代码有了，文档没有”的偏差。

中低：MySQL 迁移把 ALTER TABLE 的错误全吞了。 go/internal/storage/mysql.go:81-82 现在无条件忽略，这会把“字段已存在”和“真正迁移失败”混在一起，排障很差。应该只忽略重复列错误，其它错误要返回。

低：这两个 Go 文件还没 gofmt。 go/internal/game/model.go、go/internal/game/service_gold.go 被 gofmt -l 标出来了。不是逻辑问题，但会持续拉低代码规范度。