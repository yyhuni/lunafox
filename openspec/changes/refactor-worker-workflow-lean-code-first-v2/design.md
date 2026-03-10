## Context
当前架构已完成 code-first 主迁移，整体方向正确，但存在“可继续精简”的中间态：
- Worker 配置校验链路仍有 schema runtime + typed validate 双层实现。
- 新增 workflow 仍带有一定模板化样板（局部生成入口、资产文件约束）。

本次目标是“在不改变核心架构边界的前提下做减法”。

## Goals / Non-Goals
- Goals:
  - 降低 workflow 开发与审计心智负担。
  - 保持安全与一致性（unknown key 严格失败、版本门禁不退化）。
  - 让后续批量迁移 old-python workflows 更快。
- Non-Goals:
  - 不引入运行时动态插件。
  - 不回退到模板字符串命令执行。
  - 不改变现有 stage 执行语义和结果模型。

## Multi-Agent Review Summary
经过“架构一致性 / 开发效率 / 稳定性 / 迁移速度”多角色反复评审，结论一致：
- **不建议推翻现有 code-first 主干**。
- **最优解是 Lean Code-First v2**：保留契约单源与生成体系，去掉 Worker 冗余 runtime schema 层，补齐严格 typed decode 与自动化门禁。

## Decisions
1. 校验边界精简（核心）
- Server 继续做 schema gate（含 unknown key 与版本枚举）。
- Worker 只做：
  - 严格 typed decode（拒绝未知字段）。
  - 字段存在性检查（保持 `required` 语义，不依赖零值推断）。
  - 业务/跨字段 fail-fast 校验。
- Worker 不再执行 schema runtime 编译与校验。

2. strict decode 机制
- typed decode 路径使用 `json.Decoder` 且启用 `DisallowUnknownFields`。
- decode 流程增加字段存在性检查，覆盖 `apiVersion/schemaVersion` 及契约 required 字段。
- 保持现有错误语义映射，不放宽失败策略。

3. config_schema_runtime 收敛策略
- 在移除 Worker schema runtime 后，保留通用函数：
  - `getConfigPath`
  - `timeoutFromSeconds`
- 通用函数收敛到 `config_typed.go`，避免额外小文件分散阅读路径。

4. 生成链路精简
- 保留 contract generator。
- 增加“全局生成入口优先”（面向批量 workflow），减少每 workflow 样板命令维护。
- 入口约定：
  - 权威入口：全局生成命令（如 `make workflow-contracts-gen-all`）。
  - 增量入口：`make workflow-contracts-gen-workflow WORKFLOW=<name>`。
- 去除 per-workflow `contract_assets.go` 样板文件。
- 生成 worker schema 统一落在 `<workflow>/generated/<workflow>-<apiVersion>-<schemaVersion>.schema.json`。
- 覆盖策略：全局入口默认覆盖 `server schema/docs/typed` 目标产物，且需保证幂等。
- 维持一致性门禁：生成产物与契约必须同源、无漂移。

5. 重复校验去重
- 当前 decode 链路后在 `initialize` 中存在重复 `Validate()` 调用，本次需收敛为单次权威校验路径。
- 通过回归测试确保去重后错误语义与失败时机不变。

6. 迁移策略
- 先在 `subdomain_discovery` 落地并验证，再推广到后续 workflow。
- 使用 TDD 分阶段推进，确保每一步可回归。

## Architecture Sketch
```text
YAML
  -> Server schema gate (structural + version enum + unknown key)
  -> Scheduler compatibility gate
  -> Worker strict typed decode (unknown key reject)
  -> Worker business Validate() (cross-field fail-fast)
  -> Go stage orchestration + binary/args execution
```

## TDD Strategy
- 每个阶段遵循 Red -> Green -> Refactor：
  - Red：先写失败测试，锁定目标行为。
  - Green：最小实现使测试通过。
  - Refactor：清理重复与样板，确保可读性提升。

## Risks / Trade-offs
- 风险：移除 Worker schema runtime 后，若 strict decode/存在性检查不完整会漏掉 required 或未知字段。
  - 缓解：先写 unknown-key + required-presence 回归测试，再移除 runtime schema。
- 风险：去除 per-workflow 生成样板后，开发者可能不熟悉新入口。
  - 缓解：在 Makefile/help 与 skeleton 文档中明确全局/增量命令。
- 风险：简化过头导致业务规则下沉到工具层过多。
  - 缓解：保留最小业务规则集合（stage/tool 开关关系、关键必填字段）。

## Migration Plan
1. 先补测试覆盖（unknown key、required presence、版本字段、worker decode 行为、产物一致性）。
2. 落地 strict decode 并验证。
3. 下线 schema runtime 路径后，将通用函数收敛进 `config_typed.go`。
4. 改造生成入口为全局统一入口，并提供 Makefile 增量命令。
5. 清理重复 `Validate()` 调用并完成回归。
6. 完成全量回归与 OpenSpec 严格校验。
