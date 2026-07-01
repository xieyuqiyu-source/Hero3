# Hero3 Web

`Hero3 Web` 是 Hero3 项目的前端工程，当前基于 `React + TypeScript + Vite` 搭建，用于承载三国题材策略网页游戏的浏览器端界面、交互与后续玩法原型。

## 技术栈

- React 19
- TypeScript
- Vite
- ESLint
- pnpm

## 目录结构

```text
web/
├── public/          # 静态资源
├── src/             # 前端源码
├── index.html       # HTML 入口
├── package.json     # 前端依赖与脚本
├── vite.config.ts   # Vite 配置
└── README.md        # 前端说明文档
```

## 本地开发

安装依赖：

```bash
pnpm install
```

启动开发服务器：

```bash
pnpm dev
```

构建生产版本：

```bash
pnpm build
```

代码检查：

```bash
pnpm lint
```

本地预览构建结果：

```bash
pnpm preview
```

## 项目说明

当前前端已承载账号、城池、军事、地图、副本、万象幻境等主要玩家界面。万象幻境入口位于 `地图 -> 万象幻境`，当前包含仙池垂钓、军营豪赌和天机轮转；其中军营豪赌和天机轮转的最终结果均以后端结算响应为准，前端只负责播放动画、刷新军队和展示可兑换库存。
