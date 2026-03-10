# Change: 引入 workflow bundle-first runtime 与 builtin executor binding

## Why
当前 workflow 的运行时绑定关系仍然是隐式的：
- contract 生成 `manifest/schema/profile` 等工件；
- server/catalog/scan create 读取这些工件；
- worker 收到 `workflowId` 后直接在内置 registry 中查找同名 Go 实现执行。

这种做法在当前内置 workflow 阶段可工作，但它有两个长期问题：
- runtime 对“workflow 描述”和“workflow 执行器”之间的绑定关系缺少显式契约；
- 如果未来需要插件化或外部 workflow bundle，加一层执行器绑定模型将不可避免，而现在继续依赖隐式 `workflowId -> builtin registry` 会增加后续迁移成本。

项目已经明确未来需要插件化，因此现在最合理的中间态不是直接做完整插件系统，而是先把 runtime 演进成：
- bundle-first：运行时优先认 workflow bundle 工件；
- executor binding：bundle 显式声明该 workflow 当前由哪个执行器绑定执行；
- builtin-first：本次只落地 builtin executor，插件类型只预留模型与边界，不实现动态插件加载。

## What Changes
- 为 workflow 引入 runtime executor binding 元数据模型。
- 将 executor binding 作为 workflow runtime bundle 的一部分生成，并提供 server/worker 侧消费能力。
- 当前阶段只支持 `builtin` executor type：
  - `type = "builtin"`
  - `ref = workflowId`
- worker 执行前不再只依赖隐式 `workflowId -> registry`，而是先读取 executor binding，再解析为 builtin executor。
- server 侧 workflow bundle 语义明确化：manifest/schema/profile/executor binding 共同组成 workflow runtime bundle。
- 本次不实现 `plugin` 类型执行器，但在模型中预留：
  - `type = "plugin"`
  - `ref = ...`
- 当前变更不改变现有 workflow 执行逻辑结果，仅将隐式绑定关系提升为显式 runtime 契约。

## Impact
- Affected specs:
  - `workflow-runtime-bundle` (ADDED)
- Affected code:
  - `worker/internal/workflow/contract_types.go`
  - `worker/cmd/workflow-contract-gen/*`
  - `worker/internal/workflow/registry.go`
  - `worker/cmd/worker/main.go`
  - `server/internal/workflow/manifest/*`
  - `server/internal/modules/catalog/*`
- Behavioral impact:
  - runtime 工件不再只是 manifest/schema/profile 的松散集合，而是 bundle 语义下的组合产物；
  - worker 执行 workflow 前将通过 executor binding 确认执行器类型和引用；
  - 本次仍只支持 builtin executor，不引入动态插件加载行为。
