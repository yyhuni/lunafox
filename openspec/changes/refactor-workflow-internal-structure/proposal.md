# Change: 收敛 Workflow 内部目录结构（schema/profile 统一到 workflow 域）

## Why
当前 `server/internal/preset` 与 `server/internal/workflowschema` 已经共同服务于同一业务域（workflow catalog + scan schema gate），但目录拆分让语义与依赖边界不清晰：
- `preset` 命名落后于当前 `workflow profile` 语义。
- schema/profile 强耦合却分散在顶层目录，增加理解成本。
- worker 生成链、server import、文档路径存在硬编码，后续演进容易重复返工。

项目仍处于未上线阶段，适合进行一次性结构收敛并移除旧路径。

## What Changes
- 新建统一目录：
  - `server/internal/workflow/schema`（承载 schema 读取、metadata 列举、校验）
  - `server/internal/workflow/profile`（承载 profile 类型、加载、校验、服务）
- 统一 profile 产物目录：
  - `server/internal/preset/presets` -> `server/internal/workflow/profile/profiles`
- 迁移并替换所有 server import：
  - `internal/workflowschema` -> `internal/workflow/schema`
  - `internal/preset` -> `internal/workflow/profile`
- 迁移 worker/tooling/docs 中的路径常量：
  - `worker/scripts/gen-workflow-contracts.sh`
  - `worker/Makefile`
  - `worker/cmd/workflow-contract-gen/*`
  - `docs/workflow/contract-generation.md`
- 同步更新正在进行的 workflow 相关 OpenSpec 变更，统一引用新目录路径。
- 清理旧目录与旧路径引用，不保留兼容别名（pre-launch 一次性切换）。
- 本次仅做内部结构收敛，不改变对外 API 行为与 schema/profile 业务语义。

## Impact
- Affected specs:
  - `workflow-internal-structure`（new capability delta）
  - 与以下 change 协同：
    - `refactor-legacy-api-to-workflow-catalog`
    - `refactor-workflow-preset-generated-profiles`
- Affected code:
  - `server/internal/workflowschema/*` -> `server/internal/workflow/schema/*`
  - `server/internal/preset/*` -> `server/internal/workflow/profile/*`
  - `server/internal/preset/presets/*` -> `server/internal/workflow/profile/profiles/*`
  - `server/internal/bootstrap/*`
  - `server/internal/modules/catalog/*`
  - `server/internal/modules/scan/*`
  - `worker/scripts/gen-workflow-contracts.sh`
  - `worker/Makefile`
  - `worker/cmd/workflow-contract-gen/*`
  - `docs/workflow/contract-generation.md`
- Breaking changes:
  - **BREAKING (internal path)**: 旧目录路径删除，所有内部 import 与脚本常量必须同步迁移。
- Execution order:
  - 该变更应作为前置基线先落地，再实施 `refactor-workflow-preset-generated-profiles` 的生成产物改造。
