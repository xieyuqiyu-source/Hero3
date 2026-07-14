# NPC 页面复刻说明

## 当前版本范围

本版本按九维《武林三国》NPC 排行页复刻列表框架，并根据 Hero3 现有业务完成以下改版：

- 地图二级导航将原“NPC据点”拆为“NPC”和“据点”两个独立标签。
- NPC 不再按大型、中型、小型分页，12 座 NPC 城池统一显示在一页。
- 后端层级 `small`、`medium`、`large`、`golden` 分别显示为“小型、中型、大型、超大”，并使用灰、蓝、紫、金四种颜色区分。
- NPC 表格字段对应现有 `NpcCity` 业务模型，展示城池名称、规模、阵营、恢复特性和城池词条。
- 据点页只复用当前世界地图视野已有的黄巾据点数据，没有新增接口或修改后端。
- 副本与万象幻境仍为禁用入口，不纳入当前版本。

## 真实接口接入

- `GET /api/v1/map/npc-cities`：读取真实 NPC 列表、刷新时间和当前刷新价格。
- `POST /api/v1/map/npc-cities/refresh`：原子扣除账户金币并刷新 NPC，正式默认价格为 100 金币。
- `POST /api/v1/map/npc-cities/attack`：按 `attack` 或 `plunder` 模式即时结算所选兵力和最多一名武将。
- `POST /api/v1/map/npc-cities/scout`：由后端自动派出当前阵营侦察兵并即时结算。

页面类型、请求和过期保护分别集中在 `src/game/types.ts`、`src/api/gameApi.ts` 与 `src/npc/stateService.ts`。`src/data/npcDirectory.ts` 仅维护中文映射与排序，不再保存演示城池。

## 验收记录

- `docs/screenshots/npc-api/1280x800.png`：1280×800 真实接口数据页。
- `docs/screenshots/npc-api/1440x900.png`：1440×900 真实接口数据页。
- `npm test`：14 个测试文件、83 项测试通过。
- `npm run build`：TypeScript 校验与 Vite 生产构建通过。

本页没有引入新的官网图片资源，继续复用公共外壳和列表样式，因此无需新增图片来源条目。
