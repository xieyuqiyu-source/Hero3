**主要问题**

1. **NPC 配置是全局浅拷贝，有并发和误改风险**
   `GetNpcConfig()` 直接返回 `activeNpcCfg`，里面的 `map/slice` 还是共享引用。后续生成 NPC、GM 更新配置、读取配置并发时，容易产生数据竞争或被外部代码绕过校验改掉。
   参考：[npc.go](/Users/xieyuqiyu/Documents/Game/Hero3/go/internal/game/npc.go:102)

2. **NPC 词条目前基本只是展示，没有真正参与战斗/生产/恢复**
   配置里有 `productionBonus`、`cavalryDefenseBonus`、`armyRecoveryBonus`、`armyCapBonus` 等，但生成、结算、战斗里没有统一应用这些 buffs。这样“丰收之地”“坚毅不倒”等词条会让玩家以为有效，实际不影响结果。
   参考：[npc.json](/Users/xieyuqiyu/Documents/Game/Hero3/go/config/npc.json:44)、[service_npc.go](/Users/xieyuqiyu/Documents/Game/Hero3/go/internal/game/service_npc.go:210)、[service_combat.go](/Users/xieyuqiyu/Documents/Game/Hero3/go/internal/game/service_combat.go:407)

3. **NPC 资源写死四种资源，不跟资源字典走**
   生成 NPC 时固定 `wood/stone/iron/food`，以后如果资源系统新增资源，NPC 城池不会自动拥有新资源，也不会进入产量和仓储。
   参考：[service_npc.go](/Users/xieyuqiyu/Documents/Game/Hero3/go/internal/game/service_npc.go:212)

4. **配置校验还不够严**
   后端没校验 `refreshIntervalHours > 0`、`manualRefreshCostGold >= 0`、`count.weight/guaranteed >= 0`、`scoutCost >= 0`、恢复倍率不能为负、`cityNames` 不能为空。GM 后台保存错误配置后，可能出现每次请求都刷新 NPC、负权重、重复“未知城池”等问题。
   参考：[npc.go](/Users/xieyuqiyu/Documents/Game/Hero3/go/internal/game/npc.go:165)

5. **GM 配置界面只暴露了一部分 NPC 配置**
   后台只能改基础产量、仓储、刷新、倍率、保底数量、兵力上下限、侦查消耗；但 `armyTypes`、`traitCount`、`count.weight`、`recoveryProfiles`、`traitPool`、`cityNames` 没有可视化入口。以后调平衡还是会回到 JSON。
   参考：[NpcConfigPanel.tsx](/Users/xieyuqiyu/Documents/Game/Hero3/admin/src/components/NpcConfigPanel.tsx:142)

6. **侦查响应的前端类型和后端实际返回不一致**
   后端 `ScoutNpcResponse` 已经有 `success` 和 `battleReport`，失败时 `npcCity` 为空；但前端 `gameApi.scoutNpc` 仍声明只返回 `{ npcCity, state }`，OpenAPI 也还要求 `npcCity` 必填，缺 `success/battleReport`。
   参考：[service_combat.go](/Users/xieyuqiyu/Documents/Game/Hero3/go/internal/game/service_combat.go:321)、[game.ts](/Users/xieyuqiyu/Documents/Game/Hero3/web/src/api/game.ts:102)、[map.yaml](/Users/xieyuqiyu/Documents/Game/Hero3/docs/openapi/schemas/map.yaml:189)

7. **前端一键攻击会派出所有兵，包括侦察兵/商人等非战斗定位单位**
   `NpcCityCard` 的一键掠夺/攻击直接遍历 `army` 全部兵种，没有过滤 `role` 或可战斗能力。后端 `validateAndConsumeArmy` 也没限制单位用途。
   参考：[NpcCityCard.tsx](/Users/xieyuqiyu/Documents/Game/Hero3/web/src/pages/map/components/NpcCityCard.tsx:55)、[service_combat.go](/Users/xieyuqiyu/Documents/Game/Hero3/go/internal/game/service_combat.go:338)

8. **刷新规则现在会直接替换整批 NPC，没有保留玩家交互状态**
   自动刷新和手动刷新都是重新生成 `NpcState`，旧 NPC 的残血、被侦查、被攻击历史不会保留。MVP 可以接受，但如果后面想做“固定地图”“仇恨/占领/冷却/玩家争夺”，这个设计要升级成 NPC 独立实体或带刷新批次。
   参考：[service_npc.go](/Users/xieyuqiyu/Documents/Game/Hero3/go/internal/game/service_npc.go:31)、[service_npc.go](/Users/xieyuqiyu/Documents/Game/Hero3/go/internal/game/service_npc.go:53)

