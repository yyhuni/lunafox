# Change: 将 workflow 配置收敛为对象 canonical

## Why
当前 workflow 配置链路的核心问题不是 YAML 本身，而是“把配置当字符串在系统里传递”：

- profile 文件通过 `configuration: |` 嵌入第二段 YAML，形成 YAML-in-YAML。
- server loader / validator / catalog adapter 需要反复把字符串重新解析成对象。
- scan create、preset 合并、前端编辑器和能力识别都围绕字符串拼接或正则匹配实现，缺少结构化语义。
- 持久化层将整份配置和 task slice 存为文本，无法明确区分 canonical 配置与展示文本。

这导致以下长期问题：

- 同一配置被多次 parse / marshal，边界复杂且容易漂移。
- defaulting、overlay merge、schema 校验无法稳定围绕统一对象模型工作。
- 前端只能依赖 YAML 文本拼接和启发式解析，难以安全扩展。
- profile 产物虽然已进入生成链，但最终工件仍保留字符串式结构，继续放大历史包袱。

项目仍处于 pre-launch 阶段，适合一次性把配置 canonical 形态收敛为结构化对象，避免上线后继续维护双语义配置链路。

## What Changes
- 将 workflow 配置的 canonical 形态统一为结构化对象，而不是 YAML 字符串。
- profile 工件中的 `configuration` 改为 YAML mapping，不再使用 block scalar 包裹第二段 YAML。
- catalog / scan API 契约将 `configuration` 定义为 JSON object；YAML 只保留为导入导出或编辑器视图，不再作为主输入契约。
- scan 与 scan_task 持久化改为 JSONB canonical config；runtime gRPC 契约与 agent/worker 任务配置传递也改为对象结构，不再以 YAML 文本作为协议字段。
- preset overlay、defaulting、schema validation、workflow slice 提取统一在对象层完成，不再以字符串拼接为主。
- 前端状态改为对象 canonical；YAML 编辑器仅作为对象的序列化视图与反序列化入口。
- 本次按 pre-launch 一次性 cutover 执行，不保留长期双格式兼容链路。

## Impact
- Affected specs:
  - `workflow-config-object-canonical` (new capability delta)
  - 与以下活跃变更存在强依赖关系，需要在实施前统一基线：
    - `refactor-workflow-preset-generated-profiles`
    - `add-workflow-config-defaulting`
    - `refactor-workflow-id-semantics`
- Affected code:
  - `server/internal/workflow/profile/*`
  - `server/internal/modules/catalog/*`
  - `server/internal/modules/scan/*`
  - `server/internal/bootstrap/wiring/catalog/*`
  - `server/cmd/server/migrations/*`
  - `worker/cmd/workflow-contract-gen/*`
  - `frontend/types/*`
  - `frontend/services/*`
  - `frontend/lib/workflow-config.ts`
  - `frontend/components/scan/*`
- Operational impact:
  - **BREAKING**：workflow profile API、scan create API、scan detail API、数据库列语义都会切换到对象 canonical。
  - 需要新增数据库迁移，将旧 `yaml_configuration` / `workflow_config_yaml` 文本数据迁移到 JSONB 列。
  - runtime / worker 协议同步切到对象化；YAML 只保留为 profile 文件、编辑器视图与导入导出格式。
