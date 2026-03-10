## Context
在 `manifest/schema/profile` 工件职责收敛后，workflow 的“描述层”已经开始走向 bundle 语义，但“执行层”仍然隐式绑定在 worker builtin registry 上。未来插件化时，如果运行时仍无显式 executor binding 模型，就会出现：
- bundle 只描述 workflow 长什么样，却无法说明由谁执行；
- worker 仍然只能通过 `workflowId` 猜测 builtin executor；
- 从 builtin 向 plugin 演进时，需要再次大面积重构 server/worker 边界。

因此本次要先把 runtime bundle 和 executor binding 的契约定下来。

## Goals / Non-Goals
- Goals:
  - 引入显式 executor binding 模型；
  - 让 worker 通过 binding 而不是隐式约定解析执行器；
  - 保持当前 builtin workflow 执行行为不变；
  - 为未来 `plugin` 类型预留稳定扩展点。
- Non-Goals:
  - 本次不实现插件加载、签名校验、插件安装或沙箱；
  - 本次不改变 workflow artifact 的生成事实源；
  - 本次不做远程 registry / marketplace。

## Decisions
1. Bundle semantics
- workflow runtime bundle 逻辑上由以下工件组成：
  - manifest
  - schema
  - default/scenario profiles
  - executor binding
- 当前阶段 executor binding 先作为 manifest 顶层字段的一部分输出，减少额外 loader 改动。

2. Executor binding model
- 增加显式结构：
```json
{
  "executor": {
    "type": "builtin",
    "ref": "subdomain_discovery"
  }
}
```
- `type` 当前支持值：
  - `builtin`
  - `plugin`（仅预留，不实现）
- `ref` 对 builtin 表示 worker registry 中的 workflow key。

3. Contract ownership
- executor binding 来源于 contract definition。
- builtin workflow 必须显式声明 executor binding，而不是由生成器默认猜测。
- 这样未来迁移到 plugin 时，修改 contract 即可，无需重新设计 runtime shape。

4. Worker resolution path
- worker 启动执行 workflow 时：
  - 读取 `workflowId`
  - 读取/解析对应 executor binding
  - 若 `type == builtin`，根据 `ref` 查询 registry
  - 若未来 `type == plugin`，进入插件解析路径
- 当前 builtin 路径仍映射到现有 `workflow.Get(...)` / registry。

5. Server usage
- server/catalog 暂时不需要根据 executor binding 做 UI/行为变化，但 manifest 中应暴露该字段，供未来 catalog/runtime introspection 使用。
- scan create 当前仍只以 workflowId 驱动创建，不引入额外执行器选择参数。

## Suggested Types
Worker contract:
```go
type ContractExecutorBinding struct {
    Type string
    Ref  string
}
```

Manifest:
```go
type ManifestExecutorBinding struct {
    Type string `json:"type"`
    Ref  string `json:"ref"`
}
```

## Execution Path Sketch
```text
contract_definition.go
  -> generate manifest/schema/profile + executor binding
  -> server loads manifest bundle metadata
  -> worker receives workflowId
  -> worker resolves executor binding for workflowId
  -> builtin binding -> registry.Get(ref)
  -> execute workflow
```

## Validation Rules
- every workflow manifest MUST include executor binding
- builtin binding MUST use non-empty `ref`
- builtin binding `ref` MUST map to a registered workflow contract / registry entry
- plugin binding may exist in future, but current runtime MUST reject unsupported executor types explicitly

## Migration Strategy
- 第一步在 contract/manifest/worker 执行链引入 `builtin` binding
- 第二步让 server/catalo g可见该字段，但不要求前端立即消费
- 第三步未来再扩展 `plugin` binding type 和 loader/executor path
