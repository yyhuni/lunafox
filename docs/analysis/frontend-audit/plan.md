# Frontend 审计计划（Lunafox）

- 更新时间：2026-02-13
- 目标：在大代码量下保持“可持续、可追踪、不丢上下文”的前端质量审计流程。
- 审计基线：`web-design-guidelines` + `vercel-react-best-practices` + `react-flow-code-review` + `lint/typecheck/build/e2e`。
- 审计维度：固定执行 9 维度（排除安全维度），直到收敛。

## 1. 范围定义

- 第一优先（生产路径）：`app/[locale]` 下非 `prototypes` 页面 + `components/ui` + 业务核心组件。
- 第二优先（实验路径）：`app/[locale]/prototypes`、`components/prototypes`。
- 第三优先（设计细节）：微交互、文案一致性、视觉规范细化。

## 2. 分阶段执行

1. Phase 0（基线）
- 建立 `plan/log/issues` 三件套。
- 固化每轮审计 SOP。

2. Phase 1（静态全量）
- 按模块分片审查：`auth`、`dashboard`、`target/scan`、`settings`、`shared ui`。
- 输出 P0/P1/P2 问题清单到 `issues.csv`。

3. Phase 2（重点手审）
- 深挖高风险点：鉴权流程、复杂表单、表格交互、可访问性、URL 状态同步。
- 确认静态规则无法覆盖的逻辑问题。

4. Phase 3（动态回归）
- Playwright 关键路由巡检（中英双语）。
- 收集 `console error/pageerror/requestfailed`，形成回归报告。

5. Phase 4（收敛与闸门）
- 对未关闭问题给出明确 owner 与截止轮次。
- 将高价值检查纳入 CI（lint/typecheck/test/e2e/a11y smoke）。

## 3. 审计维度（固定 9 项）

1. 功能正确性
- 业务逻辑正确、边界条件可用、无明显功能回归。

2. 性能
- 首屏与交互性能、渲染开销、包体积与加载策略合理。

3. 标准实践
- React/Next 最佳实践、Hook 使用、状态管理与代码模式合规。

4. 可访问性（a11y）
- 键盘可达、语义化、ARIA、焦点管理与读屏友好。

5. 稳定性与容错
- 错误边界、空态/异常态、重试与超时策略可用。

6. 可维护性
- 模块边界清晰、复杂度可控、重复代码收敛、命名一致。

7. 可测试性
- 关键路径具备单测/集成/E2E 覆盖，回归可验证。

8. 可观测性
- 关键流程具备日志、埋点、错误上报与追踪能力。

9. 一致性与体验
- 设计系统一致、i18n 完整、响应式适配与交互体验一致。

## 4. 严重度定义

- P0：阻断发布 / 功能不可用 / 全局可访问性失败。
- P1：高风险缺陷 / 明显性能或交互退化。
- P2：规范偏差 / 低风险体验问题。

## 5. 完成标准（DoD）

- 每个问题有唯一 ID、文件行号、状态与验证记录。
- 已修复问题至少通过 `lint + typecheck`。
- 涉及路由与交互的修复需追加页面回归记录。
- 每轮结束必须更新 `log.md` 与 `issues.csv`。
- 每轮必须覆盖并记录 9 维度结论（至少标记 `pass/finding`）。

## 6. 每轮审计 SOP（防丢上下文）

1. 先读：`plan.md`、`log.md`、`issues.csv`。
2. 明确本轮目标与覆盖维度（9 维度中优先级最高的 2-3 项深挖，其余做基线检查）。
3. 发现问题先记台账（标注所属维度），同轮可修复则立即修复。
4. 修复后做最小验证（至少 `lint/typecheck`，涉及逻辑变更追加 `test/build`）。
5. 结束时写日志：本轮目标、维度覆盖、发现、修复、验证、下一步。

## 7. 当前状态（2026-02-13）

- 已完成：首轮重点问题修复 8 项（含 skip link、React Flow、prototype 交互语义）。
- 已完成：Round 1 生产路径静态审查（新增 4 条问题）+ Round 2 定向修复（`FE-AUD-009`~`FE-AUD-012` 全部关闭）。
- 已完成：Round 3 生产路径 i18n 收敛修复（`FE-AUD-013`~`FE-AUD-016` 全部关闭）。
- 已完成：Round 4 首轮运行时回归（单测全量通过）+ DateTimePicker 受控状态同步缺陷修复（`FE-AUD-017`）。
- 已完成：Round 5 a11y 国际化批量收敛（`FE-AUD-018`~`FE-AUD-026` 全部同轮修复关闭）。
- 已完成：Round 6 稳定性与 Hydration 收敛（`FE-AUD-027`、`FE-AUD-028` 全部同轮修复关闭）。
- 已验证：`pnpm test`（265/265）+ `pnpm build`（Round 4）通过；Round 5/6 修复后 `pnpm lint`、`pnpm typecheck` 通过。
- 已验证：Round 5 静态复扫（`aria-label` 英文硬编码、`toLocale*` 硬编码 locale、`div/span onClick`）生产路径无命中。
- 已验证：Round 6 静态复扫（`transition-all`、`autoFocus`、`github-buttons` 渲染期 `Math.random()`）命中收敛。
- 已验证：Round 7 无新增验证轮次通过（静态复扫 + `pnpm test` + `pnpm lint` + `pnpm typecheck`）。
- 已验证：Round 8 无新增验证轮次通过（规则复扫 + `pnpm build` 全路由构建成功）。
- 台账状态：当前 `issues.csv` 中 `open = 0`。
- 收敛计数：`连续无新增 P1/P2 = 2 / 2`（Round 7、Round 8 均无新增）。
- 当前结论：已满足连续无新增计数条件，进入低频巡检模式（增量改动触发新一轮）。
- 备注：浏览器级 `console error/pageerror/requestfailed` 动态巡检建议在下一轮手动/自动化回归时继续补充。
- 下一步：对后续增量变更执行“9 维度审计 -> 修复 -> 验证 -> 记账”闭环。

## 8. 自治连续执行策略（默认开启）

1. 执行模式
- 默认连续运行“审计 -> 修复 -> 验证 -> 记账”循环，不逐轮等待人工确认。
- 仅在出现阻塞时暂停并提问：缺少凭据/环境不可用、需求冲突、存在高风险破坏性变更。

2. 每轮最小动作
- 基于 skills 基线执行静态审查：`web-design-guidelines`、`vercel-react-best-practices`、`react-flow-code-review`。
- 每轮都必须输出 9 维度检查结论，并将发现的问题映射到维度。
- 发现问题先登记 `issues.csv`，同轮可修复则直接修复。
- 最少执行一次验证：`pnpm lint` + `pnpm typecheck`。
- 轮次结束必须更新 `log.md` 与 `plan.md` 当前状态。

3. 收敛退出条件
- `issues.csv` 中 P1/P2 问题为 0（或全部有明确延期说明）。
- 连续 2 轮在 9 维度下均无新增 P1/P2 问题。
- 最近一轮动态回归无 `console error` / `pageerror` / `requestfailed` 异常。

4. 持续执行硬约束
- 未满足收敛退出条件前，默认持续执行下一轮，不等待额外指令。
- 达到收敛条件后，进入“低频巡检模式”（仅增量变化触发新一轮）。
