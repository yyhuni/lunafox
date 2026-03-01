## Context
worker 当前工作流执行路径混合了模板与代码：
- Go 代码负责流程编排与结果处理。
- 模板与映射链负责命令拼接与参数对齐。
- 生成链负责 constants/schema/docs 同步。

该模式功能完整，但在“快速理解执行逻辑、快速修复问题、快速新增 workflow”方面成本较高。目标是将执行语义集中到代码中，保留配置契约与校验能力。

## Goals / Non-Goals
- Goals:
  - 让 workflow 执行逻辑在 Go 代码中单点可读。
  - 让 YAML 回归输入参数职责，避免模板 DSL 负担。
  - 保留严格校验、可观测和文档能力。
  - 为未来插件化预留边界，但不提前引入高风险运行时加载。
- Non-Goals:
  - 不在本变更中实现用户级动态插件运行时。
  - 不在上线后保留模板执行路径作为长期兼容层。
  - 不改变扫描任务编排域模型（stage/pending/blocked）的语义。

## Decisions
0. 已确认决策（2026-02-28）
- 方案 `2A`：执行命令必须采用 `binary + args[]`，迁移中必须替换现有 `sh -c` 字符串执行路径。
- 方案 `5A`：`contracts` 默认不产出；仅在“对外发布契约”场景启用可选镜像输出。
- 方案 `1A`：本次迁移范围限定为当前 worker 可执行 workflows（当前：`subdomain_discovery`）。
- 方案 `3A`：一次性切换后不保留模板执行 fallback。
- 方案 `4A`：schema/docs 采用代码生成作为唯一实现方案。
- 校验边界：采用二层最简方案，`Server=Schema 基础门禁`，`Worker=业务与跨字段权威校验（fail-fast）`。
- 错误语义：采用统一错误响应模型，固定错误码枚举，确保前后端报错语义一致。
- 兼容策略：保持 `unknown key` 严格失败，同时通过版本/能力门禁避免新旧版本混部时的前向兼容故障。

1. 执行层代码优先
- 每个 workflow 在 Go 代码中声明阶段、工具命令、重试和超时。
- 禁止运行时从模板拼接命令字符串。
- `worker/internal/activity` 必须提供标准化 `CmdRunner` 组件，workflow 代码必须通过该组件执行外部工具，不直接调用 `os/exec`。

2. 命令执行安全模型
- 命令执行接口使用 `binary + args[]` 模型。
- 禁止基于用户输入的 shell 拼接执行。
- `CmdRunner` 必须以 `context.Context` 为入口，内部统一封装 `exec.CommandContext`、超时与取消语义。
- `CmdRunner` 必须在执行前执行 `exec.LookPath(binary)` 预检并在失败时返回明确错误。
- `CmdRunner` 必须统一封装 stdout/stderr 捕获、截断日志和脱敏/安全日志输出策略。

3. 配置模型
- 每个 workflow 提供强类型 `Config` 结构与 `Validate()`。
- 配置版本字段（本次固定规范）：
  - `apiVersion`：必填，字符串，格式 `v<major>`（示例：`v1`）
  - `schemaVersion`：必填，字符串，格式 `MAJOR.MINOR.PATCH`（语义化版本示例：`1.0.0`）
- 首发约束（当前 change 范围）：
  - 当前仅 `subdomain_discovery` 在范围内。
  - 首发仅允许 `apiVersion=v1` 与 `schemaVersion=1.0.0`。
  - 新增可用版本（如 `v2` 或 `1.1.0`）必须通过新的 OpenSpec change 显式扩展，并同步更新 server schema、worker capability 与回归测试矩阵。
- Server 侧校验范围：
  - 仅做 schema 基础门禁（结构、类型、范围、required、unknown key）
  - 必须校验 `apiVersion` 与 `schemaVersion` 的“必填 + 格式”约束
  - 首发阶段必须将版本允许值固化为枚举：`apiVersion in {v1}`，`schemaVersion in {1.0.0}`
  - 不承诺覆盖全部业务语义与跨字段规则
- Worker 侧校验范围：
  - 执行 `Config.Validate()` 作为权威业务校验（含跨字段约束）
  - 对于 schema 通过但不满足业务规则的配置，按预期 fail-fast 拒绝
- 兼容性约束：
  - `unknown key` 必须直接失败，防止配置拼写错误与隐性误配置
  - 调度层必须按 `(workflow, apiVersion, schemaVersion)` 与 worker 能力做精确匹配
  - 仅当存在可用 worker 宣告支持该版本组合时才允许下发任务
  - 不兼容版本不得下发到 worker 执行，并返回 `WORKER_VERSION_INCOMPATIBLE`

4. 契约生成
- schema/docs 保留，但“单一事实源”迁移为 Go 配置定义。
- server 侧继续在创建扫描时执行 engine 配置合法性校验。
- 生成技术方案固定为“代码生成”（非手工维护）。
- “非运行时反射”定义澄清：
  - 禁止在 Server/Worker 运行期请求路径中使用反射动态生成 schema/docs。
  - 允许在 `go generate` / CI 生成阶段使用反射型工具读取 Go Struct 与 tag 产出静态 schema/docs。
  - 生成阶段产物必须是静态文件，运行期仅消费产物，不触发反射生成。
- 生成工具链约定：
  - 推荐使用 `github.com/invopop/jsonschema` 作为 schema 生成库（生成阶段可用反射）。
  - 文档生成与 schema 生成必须同源于同一组 Go 配置定义，避免漂移。
  - 不要求实现 AST 深度解析器作为首选路径。
- 契约产物目录约定：
  - 主产物：`server/internal/engineschema/*.schema.json`
  - 文档：`docs/config-reference/*.md`
  - 可选镜像：`contracts/gen/engineschema/*.schema.json`（仅用于外部分发，不参与 server 运行时校验）
- schema 文件命名约定（必须稳定且可预测）：
  - 命名格式：`<workflow>-<apiVersion>-<schemaVersion>.schema.json`
  - 首发示例：`subdomain_discovery-v1-1.0.0.schema.json`
  - 主产物与可选镜像必须使用完全相同的文件名，确保比对与分发一致。
- 生成方式约定：
  - 统一使用自动化命令生成（例如 `make workflow-contracts-gen`）。
  - CI 必须包含“生成后无差异”校验，防止手工编辑漂移。
  - 当前 `subdomain_discovery` 生成命令示例：
    - 默认（不产出 mirror）：
      `go run worker/cmd/workflow-contract-gen/main.go -workflow subdomain_discovery -worker-schema-output worker/internal/workflow/subdomain_discovery/schema_generated.json -server-schema-output server/internal/engineschema/subdomain_discovery-v1-1.0.0.schema.json -docs-output docs/config-reference/subdomain_discovery.md`
    - 启用可选 mirror（仅外部分发时）：
      `go run worker/cmd/workflow-contract-gen/main.go -workflow subdomain_discovery -worker-schema-output worker/internal/workflow/subdomain_discovery/schema_generated.json -server-schema-output server/internal/engineschema/subdomain_discovery-v1-1.0.0.schema.json -docs-output docs/config-reference/subdomain_discovery.md -mirror-schema-dir contracts/gen/engineschema`

5. 扩展策略
- 默认：仓内新增 workflow，静态注册，随版本发布。
- 未来可选：进程隔离插件协议（gRPC/STDIO/WASM）执行第三方 workflow。
- 明确不采用：worker 进程内 Go plugin `.so` 动态加载。
- 新增内置 workflow 开发模板（compile-time registration）：
  - 目录：`worker/internal/workflow/<workflow_name>/`
  - 最小文件集合：
    - `workflow.go`：实现 `workflow.Workflow` 接口并在 `init()` 中调用 `workflow.Register(...)`
    - `contract_definition.go`：声明 `WorkflowName/APIVersion/SchemaVersion` 与参数契约
    - `contract_assets.go`（仅保留生成入口与 schema embed，不承载运行时命令模板）
    - `*_test.go`：至少覆盖配置校验、阶段执行、命令构建（binary+args）
  - 生成命令：
    - `go generate ./worker/internal/workflow/<workflow_name>`
  - 回归门禁：
    - `go test ./worker/... ./server/...`
    - `openspec validate <change-id> --strict --no-interactive`
- 未来插件协议草案（仅草案，不纳入本次实现）：
  - 边界：Worker 主进程仅负责调度与资源控制，插件进程负责工作流执行。
  - 协议：推荐 gRPC/STDIO 统一 request/response，支持心跳与超时取消。
  - 最小能力：
    - `Validate(config)`：返回标准错误契约
    - `Execute(task)`：流式输出进度、结果与日志
    - `Capabilities()`：返回支持的 `(workflow, apiVersion, schemaVersion)` 元组
  - 安全约束：
    - 进程隔离、最小权限、超时强制终止、可观测审计日志
  - 明确禁止：in-process Go plugin 动态加载。

6. 错误语义与错误码
- 错误响应结构统一为：`{ code, stage, field, message }`
- `code` 枚举（本变更范围）：
  - `SCHEMA_INVALID`：Server schema 基础门禁失败（结构、类型、范围、required、unknown key）
  - `WORKFLOW_CONFIG_INVALID`：Worker `Config.Validate()` 业务/跨字段校验失败（fail-fast）
  - `WORKFLOW_PREREQ_MISSING`：Worker 运行时前置条件缺失（如 binary 不存在、必要文件不存在）
  - `WORKER_VERSION_INCOMPATIBLE`：任务配置版本与 worker 能力版本不兼容（调度阶段拦截）
- `stage` 枚举（本变更范围）：
  - `server_schema_gate`
  - `worker_validate`
  - `worker_prereq`
  - `scheduler_compatibility_gate`
- 文案规范：
  - 对外用户文案使用清晰中文，不暴露实现细节与栈信息
  - 日志保留详细调试信息，接口响应仅返回可操作提示
- 版本相关错误映射：
  - `apiVersion` / `schemaVersion` 缺失或格式不合法：`SCHEMA_INVALID` + `server_schema_gate`
  - `apiVersion` / `schemaVersion` 格式合法但不在首发允许值枚举内：`SCHEMA_INVALID` + `server_schema_gate`
  - 版本字段合法但无兼容 worker：`WORKER_VERSION_INCOMPATIBLE` + `scheduler_compatibility_gate`

## Architecture Sketch
```
YAML (user params)
  -> Server Schema Gate (structure/type/range/required)
  -> Scheduler Compatibility Gate (workflow+apiVersion+schemaVersion exact match)
  -> Worker Decode to WorkflowConfig struct
  -> Worker Validate (business + cross-field, fail-fast)
  -> Build typed execution plan in Go
  -> Execute commands via binary+args
  -> Parse output and persist results
```

## Migration Plan
1. 在上线前完成“当前 worker 可执行 workflows”（当前：`subdomain_discovery`）的 code-first 执行实现（保持输入输出契约不变）。
2. 同步切换 schema/docs 生成源为 Go 配置定义，并建立自动化代码生成与一致性校验。
3. 明确并固化“Server 基础门禁 + Worker 权威校验”的文档与错误语义边界。
4. 增加配置版本与 worker 能力版本匹配规则，避免混部发布兼容性故障。
5. 删除 template runtime 依赖路径与映射链，不保留模板执行 fallback。
6. 进行一次性全链路回归（配置校验、任务编排、执行、结果写回）后上线。

## Risks / Trade-offs
- 风险：命令变更需要发布新版本，配置侧灵活性降低。
  - 缓解：通过参数化配置保留常见可调项。
- 风险：一次性迁移改动面较大，回归压力更集中。
  - 缓解：在上线前冻结需求并完成全链路回归与冒烟清单。
- 风险：出现“Server 通过但 Worker 拒绝”时，用户感知不一致。
  - 缓解：统一错误码和报错文案，并在文档中明确这是预期 fail-fast 行为。
- 风险：Server 与 Worker 版本不一致时，新配置字段导致旧 Worker 任务失败。
  - 缓解：保持 unknown key 严格失败，并在调度阶段增加版本/能力门禁与不兼容拦截。
- 风险：若未来直接上 Go plugin，版本和安全问题会显著增加。
  - 缓解：仅考虑进程隔离插件方案，并加入签名与隔离约束。

## Open Questions
- 插件化若落地，优先 gRPC sidecar 还是 WASM runtime。
