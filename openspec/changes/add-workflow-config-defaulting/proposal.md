# Change: Workflow 配置默认值归一化与最终执行配置持久化

## Why
当前 workflow 契约已经声明了大量参数默认值，但这些默认值尚未成为真正的执行语义：

- server 侧仍以 schema 校验为主，不会先把短配置补齐为最终执行配置；
- worker 强类型解码仍要求启用工具时显式提供许多本可由契约默认值补齐的参数；
- scan 持久化的是用户原始 YAML，而不是 canonical workflow YAML；
- 如果后续只在一侧放宽短配置写法，就会出现 server / worker 行为分叉。

在 `refactor-workflow-schema-manifest-separation` 确立后，workflow 的可执行默认值语义不应继续依赖 schema 扩展字段，而应由 workflow contract / manifest 统一驱动。

## What Changes
- 将 workflow 契约中声明的参数默认值升级为真实的可执行默认值语义。
- server 在 scan create 时先基于 workflow contract / manifest 规范化 workflow 配置，再用 workflow schema 校验规范化结果，并持久化 canonical workflow YAML。
- worker 在强类型解码前应用与 server 一致的默认值补齐语义，避免入口差异导致行为漂移。
- worker 保留执行前校验职责，但不再重复要求显式提供可由默认值补齐的参数。
- generator 继续输出 workflow schema 中的标准 `default` 注解以服务通用工具与文档，但执行语义事实源保持为 contract / manifest，而不是 schema 扩展字段。
- 保持显式行为开关原则：不自动补 `stage.enabled`、`tool.enabled`，不替用户自动开启扫描动作。

## Impact
- Affected specs:
  - `workflow-config-defaulting` (ADDED)
- Affected code:
  - `/Users/yangyang/Desktop/lunafox/worker/cmd/workflow-contract-gen/main.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/config_typed.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/workflow_metadata.go`
  - `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_execute.go`
  - `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_plan.go`
  - generated schema / manifest / profile / docs 产物
- Behavioral impact:
  - 新建 scan 的 `yaml_configuration` 将从“用户原始 YAML”变为“系统归一化后的 canonical workflow YAML”。
  - 允许启用工具时省略带默认值的参数，但仍要求显式提供 stage / tool / `enabled` 结构。
  - 默认值补齐与 canonical config 归一化不再依赖 schema 扩展字段。
