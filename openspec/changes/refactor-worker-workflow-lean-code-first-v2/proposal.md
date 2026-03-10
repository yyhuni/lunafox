# Change: Worker Workflow Lean Code-First v2（简化校验与生成链路）

## Why
当前 code-first 底座已经稳定，但仍有两类“中度复杂度”问题影响长期扩展：
- Worker 侧存在 schema runtime 校验与 typed 校验双路径，理解成本偏高。
- 每个 workflow 仍需维护局部生成入口与资产文件，批量迁移 old-python workflows 时样板较多。

项目仍处开发阶段、未上线，适合在此时做一次“结构不推翻、复杂度下调”的精简改造。

## What Changes
- 保持 `contract_definition.go` 为契约单一事实源，不改回 YAML 契约源。
- 收敛校验分层：
  - Server 保留 schema 基础门禁（结构/类型/required/unknown key/版本枚举）。
  - Worker 移除 schema runtime 编译校验，改为严格 typed decode + 业务规则校验。
- 在 Worker typed decode 中引入严格 unknown-key 拒绝（`DisallowUnknownFields` 语义），避免移除 schema runtime 后丢失防护。
- 补齐 Worker 侧 `required` 存在性语义（避免仅靠零值导致缺字段静默通过）。
- 统一生成链路到“全局生成入口”，减少每个 workflow 的重复生成样板和局部生成文件。
- 去除 per-workflow `contract_assets.go` 样板，按目录动态发现 workflow 并生成。
- 将 `config_schema_runtime` 相关通用函数收敛到 `config_typed.go`，避免额外小文件分散认知。
- 清理 decode 后重复 `Validate()` 调用，消除重复校验路径。
- 用 TDD 分阶段执行（Red→Green→Refactor），确保简化过程中无行为回归。

## Impact
- Affected specs:
  - `worker-workflow-code-first-execution` (MODIFIED)
- Affected code:
  - `worker/internal/workflow/subdomain_discovery/config_schema_runtime.go`（预计下线或最小化）
  - `worker/internal/workflow/subdomain_discovery/config_typed.go`
  - `worker/internal/workflow/subdomain_discovery/workflow.go`
  - `worker/cmd/workflow-contract-gen/*`
  - `worker/Makefile`
  - `worker/internal/workflow/*`（生成与一致性门禁）
- Operational impact:
  - 行为上保持“Server gate + Worker fail-fast”。
  - 目标是减少维护复杂度，不改变扫描语义与结果模型。
