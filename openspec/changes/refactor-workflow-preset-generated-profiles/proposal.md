# Change: Workflow 预设改为“契约生成 + 可选策略覆盖”

## Why
当前项目在 workflow 契约层已经实现 code-first 与自动生成（schema/docs/typed config），但 preset 默认配置仍由 `server/internal/preset/presets/*.yaml` 手工维护。这导致以下问题：
- 契约与预设存在“双真相源”，容易发生漂移。
- 预设更新依赖人工同步，难以保证版本一致与可回溯。
- 关于默认值与场景策略（fast/deep）的语义边界不清晰，影响后续演进。

项目仍处于上线前阶段，适合在此时完成默认值治理收敛，降低上线后演进成本。

## What Changes
- 引入“生成型 preset”模型：
  - `default` 预设由 workflow 契约源自动生成（不再手写完整 YAML）。
  - 场景策略（如 `fast`/`deep`）使用可选 overlay（仅定义差异字段），由生成链合并产出最终配置。
- 扩展契约生成工具链：
  - 在现有 `workflow-contract-gen`/脚本链路中新增 preset 产物生成。
  - 生成目标目录固定为 `server/internal/preset/presets`。
  - 生成后必须执行 schema 校验门禁，失败即中断。
- 一次性切换（未上线阶段）：
  - 本次改造按 pre-launch 一次性 cutover 执行，不保留旧 preset 兼容路径。
  - 旧 preset 命名/旧手工文件作为历史实现清理，不再承诺兼容读取。
  - `preset.Loader` 继续消费最终产物文件，不引入运行时动态生成。
- 增加 CI 漂移防护：
  - 生成命令纳入 CI “无差异”检查，禁止手工修改生成产物而未同步更新源定义。

## Impact
- Affected specs:
  - `workflow-preset-generated-profiles` (new capability delta)
- Affected code:
  - `worker/cmd/workflow-contract-gen/*`
  - `worker/scripts/gen-workflow-contracts.sh`
  - `worker/internal/workflow/*/contract_definition.go`（默认值与 profile 源定义）
  - `server/internal/preset/*`
  - `server/internal/modules/catalog/*`（preset ID/响应语义调整）
  - `frontend/*`（preset 选择与 ID 依赖链路同步）
- Operational impact:
  - 上线前需要一次性迁移现有手工 preset 到生成源。
  - 迁移后 preset 主文件视为生成产物，不建议手工编辑。
  - **BREAKING**：不保留旧 preset 兼容层，需同步前后端与测试基线。
