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
- YAML 解析后必须进行：
  - 结构校验（类型、必填、范围）
  - 未知字段校验（unknown key 直接失败）
  - 跨字段约束校验（例如 stage/tool enabled 时必填参数）

4. 契约生成
- schema/docs 保留，但“单一事实源”迁移为 Go 配置定义。
- server 侧继续在创建扫描时执行 engine 配置合法性校验。
- 生成技术方案固定为“代码生成”（非运行时反射、非手工维护）。
- 契约产物目录约定：
  - 主产物：`server/internal/engineschema/*.schema.json`
  - 文档：`docs/config-reference/*.md`
  - 可选镜像：`contracts/gen/engineschema/*.schema.json`（仅用于外部分发，不参与 server 运行时校验）
- 生成方式约定：
  - 统一使用自动化命令生成（例如 `make workflow-contracts-gen`）。
  - CI 必须包含“生成后无差异”校验，防止手工编辑漂移。

5. 扩展策略
- 默认：仓内新增 workflow，静态注册，随版本发布。
- 未来可选：进程隔离插件协议（gRPC/STDIO/WASM）执行第三方 workflow。
- 明确不采用：worker 进程内 Go plugin `.so` 动态加载。

## Architecture Sketch
```
YAML (user params)
  -> Decode to WorkflowConfig struct
  -> Validate struct + business constraints
  -> Build typed execution plan in Go
  -> Execute commands via binary+args
  -> Parse output and persist results
```

## Migration Plan
1. 在上线前完成“当前 worker 可执行 workflows”（当前：`subdomain_discovery`）的 code-first 执行实现（保持输入输出契约不变）。
2. 同步切换 schema/docs 生成源为 Go 配置定义，并建立自动化代码生成与一致性校验。
3. 删除 template runtime 依赖路径与映射链，不保留模板执行 fallback。
4. 进行一次性全链路回归（配置校验、任务编排、执行、结果写回）后上线。

## Risks / Trade-offs
- 风险：命令变更需要发布新版本，配置侧灵活性降低。
  - 缓解：通过参数化配置保留常见可调项。
- 风险：一次性迁移改动面较大，回归压力更集中。
  - 缓解：在上线前冻结需求并完成全链路回归与冒烟清单。
- 风险：若未来直接上 Go plugin，版本和安全问题会显著增加。
  - 缓解：仅考虑进程隔离插件方案，并加入签名与隔离约束。

## Open Questions
- 插件化若落地，优先 gRPC sidecar 还是 WASM runtime。
