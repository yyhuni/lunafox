# Scan History Runtime Drawer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 scan history 右侧抽屉改造成运行控制 / 排障视角长页面，突出任务状态与任务日志。

**Architecture:** 保留列表入口与抽屉容器，重写抽屉内部布局。以 `stageProgress` 推导任务状态与 demo 日志内容，复用现有 scan logs/config 展示能力，同时将资产信息降级为轻量摘要区。

**Tech Stack:** Next.js, React, TypeScript, next-intl, shadcn/ui Sheet, Tailwind CSS.

---

### Task 1: 重写抽屉布局
- Modify: `frontend/components/scan/history/scan-runtime-detail-drawer.tsx`
- Replace dashboard-like summary with runtime-focused sections

### Task 2: 强化任务状态与任务日志
- Modify: `frontend/components/scan/history/scan-runtime-detail-drawer.tsx`
- Implement expandable task items with inline task log panels

### Task 3: 调整文案与入口
- Modify: `frontend/components/scan/history/scan-history-columns.tsx`
- Modify: `frontend/messages/zh.json`
- Modify: `frontend/messages/en.json`

### Task 4: 验证
- Run: `pnpm -C frontend exec eslint ...`
- Run: `pnpm -C frontend build`
