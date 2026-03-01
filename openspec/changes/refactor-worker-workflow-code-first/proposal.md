# Change: Worker 工作流改为代码优先执行模型

## Why
当前 worker 工作流执行同时依赖 Go 代码、YAML 模板、参数映射与生成链路，虽然具备灵活性，但在日常开发中出现以下问题：
- 可读性不足：执行命令、参数来源、阶段依赖分散在多处，排障与评审成本高。
- 维护复杂：`semantic_id/config_schema.key/var` 多层映射增加认知负担与变更风险。
- 扩展门槛高：新增 workflow 需要同时理解模板与代码路径，不利于团队持续交付。

项目当前更需要“可维护、可调试、可演进”的执行模型，因此将 workflow 执行层收敛为代码优先。

## What Changes
- 将 worker workflow 的执行层统一为 Go 硬编码实现：
  - 阶段编排、命令定义、参数装配、重试与超时策略均在 Go 代码中定义。
  - 不再在运行时使用模板驱动命令构建与参数 key 映射。
  - 命令执行统一为 `binary + args[]`，不保留 `sh -c` 字符串执行路径。
  - 引入 `worker/internal/activity` 标准化 `CmdRunner` 组件，统一承载命令执行、取消/超时传播、输出捕获、二进制存在性预检。
- 将前端 YAML 的职责收敛为“输入参数”：
  - YAML 只描述用户可调配置，不再描述命令模板。
  - 解析后映射到每个 workflow 的强类型 `Config` 结构，并执行严格校验。
- 保留并重建“配置契约能力”：
  - schema 与文档继续提供，但来源由模板迁移为 Go 配置定义（或生成规则）。
  - server 在创建扫描时仅执行 schema 基础门禁校验（结构、类型、范围、required）。
  - worker 在解析配置后执行权威业务校验（含跨字段约束），并以 fail-fast 方式拒绝不满足业务规则的配置。
  - 若出现“server 通过但 worker 拒绝”，视为预期行为，不再追求将全部业务规则塞入 schema。
  - `unknown key` 保持严格失败，不采用“仅告警放行”策略。
  - 为避免新旧版本混部导致失败，增加版本/能力门禁调度：
    - 配置中显式包含 `apiVersion`（`v<major>`）和 `schemaVersion`（`MAJOR.MINOR.PATCH`）。
    - 首发固定值：`apiVersion=v1`、`schemaVersion=1.0.0`（以 schema 枚举强约束）。
    - 超出首发允许值的配置在 Server 侧直接拒绝，不进入执行链路。
    - 调度阶段按 `(workflow, apiVersion, schemaVersion)` 与 worker capability 做精确匹配，不兼容直接拦截。
  - 统一错误响应结构与错误码枚举，保证前后端在二层校验下的报错语义一致。
  - 生成产物主目录固定为：
    - schema: `server/internal/engineschema/*.schema.json`
    - docs: `docs/config-reference/*.md`
  - `contracts/` 默认不产出；仅在对外分发场景启用可选镜像，不作为运行时校验主来源。
- 约定扩展路径：
  - 默认扩展方式为“新增 Go workflow + 静态注册 + 发布”。
  - 未来若支持用户插件，采用进程隔离协议（例如 gRPC/STDIO/WASM），不采用 worker 进程内 Go plugin 动态加载。
- 采用未上线阶段的一次性迁移策略：
  - 在正式上线前完成“当前 worker 可执行 workflows”的 code-first 切换（当前范围：`subdomain_discovery`）。
  - 不引入长期双模型兼容窗口，不保留模板执行路径。
- 约定自动化生成入口：
  - 使用统一脚本/命令自动生成 schema/docs，避免手工维护。
  - 生成命令纳入 CI 校验，确保产物与代码定义一致。

## Impact
- Affected specs:
  - `worker-workflow-code-first-execution` (new capability delta)
- Affected code:
  - `worker/internal/workflow/subdomain_discovery/*`（本次范围）
  - `worker/internal/activity/*`（模板加载与命令构建相关路径将收敛）
  - `worker/cmd/*`（生成工具入口将统一）
  - `server/internal/engineschema/*`
  - `server/internal/modules/scan/application/*`（配置校验与任务规划对接）
  - `docs/config-reference/*`
  - `contracts/gen/*`（仅在启用镜像输出时）
- Operational impact:
  - 上线前需要完成一次性切换与回归验证。
  - 上线后仅保留 code-first 执行模型，降低长期维护复杂度。
  - 命令变更将通过代码发布流程进行管理。
