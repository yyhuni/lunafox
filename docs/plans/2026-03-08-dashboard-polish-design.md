# Dashboard Professional Polish Design

**日期：** 2026-03-08

## 目标

将 `/zh/dashboard/` 从偏展示型的 Bauhaus demo 风格，重塑为更成熟、更可信、更适合安全平台首页的企业级控制台首页，同时保持现有信息架构与数据来源不变。

## 设计方向

- 保留当前三层结构：总览、态势分析、运行任务
- 弱化演示型视觉符号与过度风格化标签
- 强化运营视角：让用户优先看到当前状态、风险趋势与任务运行情况
- 中文界面统一中文业务语义，仅保留必要专有名词

## 页面结构

### 1. 头部
- 从“风格化标题条”改为“安全运营总览”头部
- 左侧提供正式标题与一句简短说明
- 右侧保留环境状态与最近刷新时间
- 去除 `DASH-01`、过强的编号/实验感元素

### 2. 指标卡
- 改为正式的运营指标卡
- 统一结构：指标名、核心数字、变化趋势、辅助说明
- 数字优先，图标退居辅助角色
- 变化 badge 更克制，减少视觉抢占

### 3. 分析面板
- 趋势图与漏洞分布统一为分析卡片体系
- 标题、副标题、图例和间距统一
- 用更强的中文业务标题替换展示型英文 kicker
- 让图表传达“当前态势”，而非“设计组件演示”

### 4. 运行任务表格
- 首页表格定位为“最近运行任务”总览
- 强化列主次关系，降低标签噪音
- 状态表达更正式，便于一眼判断
- 首页仅承担总览职责，不承载详情页全部信息密度

### 5. 全局视觉策略
- 提高主内容层级，降低侧栏干扰
- 增加卡片与区块之间的层级差
- 文案、状态、字号、边框、留白与分隔统一
- 保持现有主题机制兼容，但 dashboard 首页优先呈现专业产品感

## 实现边界

- 不变更路由、接口、数据模型
- 不引入新的复杂业务交互
- 不重做左侧导航架构
- 仅改造 dashboard 首页相关前端组件与文案

## 涉及文件

- `frontend/app/[locale]/dashboard/page.tsx`
- `frontend/components/dashboard/bauhaus-dashboard-header.tsx`
- `frontend/components/dashboard/dashboard-stat-cards.tsx`
- `frontend/components/dashboard/asset-trend-chart.tsx`
- `frontend/components/dashboard/vuln-severity-chart.tsx`
- `frontend/components/dashboard/dashboard-lazy-sections.tsx`
- `frontend/components/dashboard/dashboard-data-table.tsx`
- `frontend/components/dashboard/dashboard-scan-history.tsx`
- `frontend/messages/zh.json`

## 验证方式

- 组件测试验证关键文案与结构输出
- 本地页面回看 `/zh/dashboard/`
- 检查首页层级、文案统一性、状态表达与控制台噪音
