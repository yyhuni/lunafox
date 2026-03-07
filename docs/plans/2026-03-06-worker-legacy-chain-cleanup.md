# 2026-03-06 Worker Legacy Chain Cleanup Plan

## Goals
- 清除误导性的 legacy worker frontend HTTP 链路
- 保持 `/settings/workers/` 继续走 agent 主线
- 去掉未接通的 worker deploy placeholder 对运行时契约的误导

## Planned steps
1. 删除 `frontend/services/worker.service.ts` 与 `frontend/hooks/use-workers.ts`
2. 删除只服务于该链路的测试与 mock
3. 处理 `frontend/types/worker.types.ts` 及 deploy-terminal 占位 UI
4. 跑 workers settings / agents 定点测试
5. 做全文搜索和 OpenSpec 校验
