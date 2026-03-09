# Dashboard Professional Polish Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 dashboard 首页重构为更专业、更统一的安全运营控制台首页，同时保持现有数据流与页面骨架稳定。

**Architecture:** 通过局部重构 dashboard 头部、指标卡、分析面板、表格容器与中文文案，提升首页的视觉层级与业务表达，不修改后端接口与核心业务逻辑。优先在组件层完成结构与样式收敛，并用前端组件测试锁定新的标题、说明与关键文案输出。

**Tech Stack:** Next.js 15、React 19、Tailwind CSS、next-intl、Vitest、Testing Library、Recharts、framer-motion

---

### Task 1: 为专业化文案建立测试基线

**Files:**
- Create: `frontend/components/dashboard/__tests__/dashboard-professional-polish.test.tsx`
- Modify: `frontend/messages/zh.json`

**Step 1: Write the failing test**
- 为 dashboard 头部、指标卡、分析面板、任务区标题写渲染断言
- 断言新的专业化中文文案存在，旧的演示型文案不再作为主标题出现

**Step 2: Run test to verify it fails**
- Run: `pnpm test -- dashboard-professional-polish.test.tsx`
- Expected: FAIL，因新文案和新结构尚未实现

**Step 3: Write minimal implementation**
- 在 `frontend/messages/zh.json` 中补充新的 dashboard 文案键位
- 如需要，为组件暴露更稳定的标题和说明文本

**Step 4: Run test to verify it passes**
- Run: `pnpm test -- dashboard-professional-polish.test.tsx`
- Expected: PASS

### Task 2: 重构 dashboard 头部与页面骨架

**Files:**
- Modify: `frontend/app/[locale]/dashboard/page.tsx`
- Modify: `frontend/components/dashboard/bauhaus-dashboard-header.tsx`
- Modify: `frontend/components/dashboard/dashboard-lazy-sections.tsx`

**Step 1: Add or extend failing test assertions**
- 断言头部标题、副标题、状态区文案与概览结构

**Step 2: Run test to verify it fails**
- Run: `pnpm test -- dashboard-professional-polish.test.tsx`
- Expected: FAIL

**Step 3: Implement minimal structure update**
- 将头部改为正式运营总览头部
- 调整页面主容器的间距与区块节奏

**Step 4: Run test to verify it passes**
- Run: `pnpm test -- dashboard-professional-polish.test.tsx`
- Expected: PASS

### Task 3: 重构指标卡与分析面板表达

**Files:**
- Modify: `frontend/components/dashboard/dashboard-stat-cards.tsx`
- Modify: `frontend/components/dashboard/asset-trend-chart.tsx`
- Modify: `frontend/components/dashboard/vuln-severity-chart.tsx`

**Step 1: Extend failing test assertions**
- 断言新的模块标题、副标题、说明和关键业务标签

**Step 2: Run test to verify it fails**
- Run: `pnpm test -- dashboard-professional-polish.test.tsx`
- Expected: FAIL

**Step 3: Implement minimal UI update**
- 调整卡片结构与标题说明
- 去除/弱化展示型英文 kicker
- 统一趋势与风险面板的标题和容器表现

**Step 4: Run test to verify it passes**
- Run: `pnpm test -- dashboard-professional-polish.test.tsx`
- Expected: PASS

### Task 4: 收敛任务表格总览语义

**Files:**
- Modify: `frontend/components/dashboard/dashboard-data-table.tsx`
- Modify: `frontend/components/dashboard/dashboard-scan-history.tsx`

**Step 1: Extend failing test assertions**
- 断言任务区标题、说明、状态概览文案存在

**Step 2: Run test to verify it fails**
- Run: `pnpm test -- dashboard-professional-polish.test.tsx`
- Expected: FAIL

**Step 3: Implement minimal UI update**
- 为首页表格增加总览标题与说明
- 调整信息密度与视觉层级

**Step 4: Run test to verify it passes**
- Run: `pnpm test -- dashboard-professional-polish.test.tsx`
- Expected: PASS

### Task 5: 完整验证

**Files:**
- Verify: `frontend/components/dashboard/**`
- Verify: `frontend/messages/zh.json`

**Step 1: Run focused tests**
- Run: `pnpm test -- dashboard-professional-polish.test.tsx`

**Step 2: Run type and lint verification**
- Run: `pnpm typecheck`
- Run: `pnpm lint`

**Step 3: Run visual verification**
- 打开 `http://localhost:3000/zh/dashboard/` 复查页面层级、文案与交互状态
