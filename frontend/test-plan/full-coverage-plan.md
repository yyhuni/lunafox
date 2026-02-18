# Frontend 完整覆盖计划

## 目标
- 覆盖范围：`按钮`、`功能链路`、`边界输入`、`危险操作`。
- 执行方式：可续跑、可并发、可回写状态，直到达到完成标准。

## 完成标准 (Definition of Done)
1. 路由巡检：计划路由全部通过（`routes.todo.json` 全部 `passed`）。
2. 交互清单：每个页面可见交互控件已盘点（`interaction-matrix.json`）。
3. 边界清单：每个输入控件都有边界测试项（`boundary-cases.todo.json`）。
4. 按钮断言：关键按钮完成断言型 e2e（触发后结果可验证，而非仅点击）。
5. 危险操作：删除/停止/重置等在安全环境验证，含回滚策略。
6. 最终回归：全量并发巡检通过，无 blocker。

## 状态文件
- `/Users/yangyang/Desktop/lunafox/frontend/test-plan/full-coverage.todo.json`
  - 阶段级状态机（`pending/running/passed/failed`）。
- `/Users/yangyang/Desktop/lunafox/frontend/test-plan/full-coverage-progress.md`
  - 每轮执行日志。
- `/Users/yangyang/Desktop/lunafox/frontend/test-plan/routes.todo.json`
  - 路由级覆盖状态。
- `/Users/yangyang/Desktop/lunafox/frontend/test-plan/interaction-matrix.json`
  - 页面交互控件清单。
- `/Users/yangyang/Desktop/lunafox/frontend/test-plan/boundary-cases.todo.json`
  - 边界测试任务清单。

## 执行命令
```bash
cd /Users/yangyang/Desktop/lunafox/frontend
pnpm run test:e2e:full:loop
```

## 并发参数
- `SMOKE_CONCURRENCY`：路由巡检并发（默认 `4`）。
- `SMOKE_RETRY_CONCURRENCY`：失败重试并发（默认 `2`）。
- `MATRIX_CONCURRENCY`：交互清单采集并发（默认 `3`）。
- `FULL_MAX_ROUNDS`：每阶段最大重试轮数（默认 `8`）。

示例：
```bash
cd /Users/yangyang/Desktop/lunafox/frontend
SMOKE_CONCURRENCY=6 SMOKE_RETRY_CONCURRENCY=2 MATRIX_CONCURRENCY=4 FULL_MAX_ROUNDS=10 pnpm run test:e2e:full:loop
```

## 续跑规则
1. 每次从 `full-coverage.todo.json` 读取阶段状态继续。
2. 自动阶段失败则记录并重试；超过上限标记 `failed`。
3. 手工阶段保持 `pending`，逐项推进后改为 `passed`。
4. 每轮执行都必须写入 `full-coverage-progress.md`。

## 危险操作策略
- 默认不自动触发危险操作。
- 仅在 mock 或隔离数据环境执行。
- 每个危险操作必须记录：前置数据、操作步骤、回滚方式、验证结果。

## 续跑口令
```text
读取 /Users/yangyang/Desktop/lunafox/frontend/test-plan/full-coverage.todo.json 与 full-coverage-progress.md，
按 pending/failed 阶段继续执行，完成一个阶段就回写状态，直到全部阶段 passed 再汇报。
```
