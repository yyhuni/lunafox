# Scan History Runtime Drawer Design

**日期：** 2026-03-09

## 目标

将 `/[locale]/scan/history/` 里的运行详情抽屉重构为更偏运行控制 / 排障视角的长页面，突出 scan 状态、任务状态、任务日志、扫描配置与整体扫描日志，而不是传统 dashboard 式总览。

## 设计方向

- 不再突出“大进度概览”与资产统计卡片
- 顶部只保留必要的 scan 状态与运行元信息
- 抽屉主体改为任务状态清单，每个任务可展开查看日志与失败信息
- 扫描配置与整体扫描日志保留为独立区块
- 资产结果仅做轻量展示，不做大卡片
- 失败信息下沉到任务状态与任务日志，不单独做 scan 级大失败 Hero

## 页面结构

1. 运行元信息
- scan 状态
- scan ID
- 目标名称
- worker / 节点
- 创建时间 / 结束时间

2. 任务状态清单（主体）
- 每个 task 一行
- 展示 workflow / stage / 状态 / 失败标签
- 支持展开查看任务日志和详细说明

3. 扫描配置
- 只读 YAML / 配置展示

4. 整体扫描日志
- 复用现有 scan log panel

5. 轻量资产展示
- 以紧凑统计行/标签展示已发现资产

## 实现边界

- 本次仍然不引入真实 task 日志 API
- 任务日志 demo 由 `stageProgress` 中的 `detail/error/reason` 与 scan failure 信息推导
- 仍然保留现有 scan progress dialog，不影响其它入口
- 仅重写 scan history 抽屉，不改 scan detail 页面路由

## 涉及文件

- `frontend/components/scan/history/scan-runtime-detail-drawer.tsx`
- `frontend/components/scan/history/scan-history-list-state.ts`
- `frontend/components/scan/history/scan-history-list-view-state.ts`
- `frontend/components/scan/history/scan-history-columns.tsx`
- `frontend/components/scan/history/scan-history-list-sections.tsx`
- `frontend/messages/zh.json`
- `frontend/messages/en.json`

## 验证方式

- 定向 eslint 检查改动文件通过
- `pnpm -C frontend build` 通过编译与类型检查；若最终 page data 收集被仓库既有问题阻断，则记录为非本次问题
