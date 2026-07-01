Original prompt: [天机轮转老虎机系统设计.md](/Users/xieyuqiyu/Documents/Game/Hero3/docs/系统设计/天机轮转老虎机系统设计.md) 文档更新了  请按照文档继续开发。

## 2026-07-01

- 已确认设计文档第二版要求：3x3 服务端权威 grid、固定 5 条赔付线、lineBet 总押注、Wild、Scatter 免费旋转、Bonus 奖励局、库存兑换和 GM 记录。
- 当前代码仍是上一版 3 个符号单线结算，需要替换后端结算、前端展示、接口类型、OpenAPI 和测试。
- 已完成后端第二版核心结算：固定 5 线、Wild、Scatter 免费旋转、Bonus、单条记录库存沉淀、奖励上限保护。
- 已完成前端第二版主界面：每线押注、总押注预览、3 个竖向转轮各 3 格、按后端 grid 停靠、免费旋转自动播放、结果明细。
- 已执行 `cd web && npm run build`，前端构建通过。
- 已执行 `make openapi`、`go test ./internal/app/game -run 'Test.*Slot|Test.*MiniGame' -count=1`、`go test ./internal/app/game -count=1`、`go test ./internal/infrastructure/storage -count=1`、`go test ./internal/transport/api -count=1`、`cd admin && npm run build`、`git diff --check`，均通过。
- Playwright 技能脚本因当前环境缺少可被 ESM 直接解析的 `playwright` 包未能运行；已用 Playwright MCP 兜底加载本地 Vite 页面并截图检查，页面无白屏和无 console/pageerror。当前登录存档兵力不足 100,000，天机轮转正确显示不可押注空态。
- 用户最新要求去掉押注数量限制：已改为不再使用 `maxLineBet/maxTotalBet/maxBetRatio`，天机轮转只要求每线押注达到最小值且总押注不超过当前拥有兵力。
- 用户要求老虎机更像真实滚轴：已把天机轮转动画改为竖向 strip 滑动，左/中/右三列放慢并依次停轮，停轮前减速，落点轻微回弹；已用 Playwright MCP 拦截假结算验证滚动中和结果态截图。
- 用户继续反馈“慢慢停下来后内容变了、滚动看不清、没有紧张感”：已把滚轴长条末尾固定为后端最终 3x3 grid，避免停轮后随机换图；滚轴加长到 14 行填充，三列按约 2.6s、4.0s、5.5s 依次停稳，前段逐格滑动、后段减速停靠。已用 Playwright 代码执行工具拦截假结算，按 0.45s、2.95s、4.55s、6.45s 截图检查，确认 0/3、1/3、2/3、3/3 锁轮过程和最终 grid 一致，控制台无 error。
- 用户反馈停轮有抖动、规则不清楚、中奖反馈不明显：已移除停轮容器回弹动画和末端 overshoot；新增“查看天机轮转玩法”说明弹窗；中奖格按 `winningLines.positions` 加 `animate-slot-win-flash` 快速闪烁约 3 秒后再弹结算；未中奖弹窗改为“很遗憾，未中奖”和“再来一局”。验证：`cd web && npm run build` 通过；Playwright 验证规则弹窗文案、中奖 3 格闪烁且弹窗延后、未中奖弹窗文案，控制台无 error。
- 用户要求不同符号增加小图片：已新增 `web/src/assets/minigames/slot/` 下 9 个 SVG 小图标，分别对应玄铜符、白银符、赤金符、玉玺、虎符、天命令、天机令、星陨、宝匣；`SlotMachineGame.tsx` 滚轴格和结果弹窗矩阵已统一显示这些图标。验证：`cd web && npm run build` 通过；Playwright 截图检查待机滚轴与中奖结果弹窗图标清晰，控制台无 error。
