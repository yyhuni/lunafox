# Application Port Standard

本文档定义 `internal/modules/*/application` 的端口（port）组织规范，用于统一接口放置与命名方式。

## 目标

- 保持应用层依赖倒置：应用层声明能力，基础设施层提供实现。
- 统一术语与文件模式，降低跨模块迁移与检索成本。
- 在统一风格的同时，避免一次性大改造成风险。

## 核心策略（混合策略）

- **单用例接口就近定义**：仅被单个服务/文件使用的接口，可放在对应 `*_query.go`、`*_command.go` 或用例文件内。
- **跨用例复用接口集中定义**：被多个应用层文件复用的接口，放在资源化命名的 `*_ports.go` 文件内（禁止泛名 `ports.go`）。

## 命名规范

- 查询能力接口：`XxxQueryStore`
- 写入能力接口：`XxxCommandStore`
- 组合接口：`XxxStore`（通过 interface embedding 组合 Query/Command）

示例（agent 模块）：

- `AgentQueryStore`
- `AgentCommandStore`
- `AgentStore`

## `*_ports.go` 边界

`*_ports.go` 允许：

- 接口定义与接口组合
- 应用层错误别名（`var ErrXxx = ...`）
- 应用层类型别名（`type Xxx = ...`）

`*_ports.go` 禁止：

- Service 结构体定义
- 业务流程函数实现
- 基础设施实现细节（如 GORM 查询/更新逻辑）
- 与应用层端口无关的默认实现代码

## 文件命名约束

- 新代码不再使用 `contracts.go`，统一使用资源化 `*_ports.go`。
- 禁止使用泛名文件：`ports.go`、`commands.go`、`service.go`、`types_alias.go`。
- 端口文件名应体现资源与职责，例如：`scan_create_ports.go`、`task_runtime_query_ports.go`。

## 渐进迁移准则

- 先改活跃模块，避免全仓一次性迁移。
- 每次迁移仅做接口组织与依赖收敛，不改变业务行为。
- 迁移后至少通过：
  - 模块 application 单测
  - `check-naming-conventions`
  - `check-layer-dependencies`
