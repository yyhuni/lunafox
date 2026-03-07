# 2026-03-06 Worker Legacy Chain Cleanup Design

## Background
`/settings/workers/` 已经切到 agent 主线，但仓库里还残留一条旧的 frontend worker service/hook/types 链路，以及一条未接通的 deploy-terminal 占位 UI 子链。这些残留代码持续制造两个问题：
- 容易让人误以为 `/workers` 和 `/ws/workers/:id/deploy/` 仍然是受支持的后端契约
- 在命名治理上保留了与主线相冲突的 snake_case 和旧 DTO 假设

## Recommended approach
采用“删除遗留链路优先”的标准方案：
- 保留 agent 主线，不新增 `/workers` 后端契约
- 删除 legacy `/workers` service / hook / test
- 删除或隔离未接通的 worker-only deploy terminal UI
- 用定点测试和全文搜索证明 workers 页面仍然走 agent 契约

## Scope
首轮仅处理前端遗留链路，不引入新的 backend worker API / WS 能力。
