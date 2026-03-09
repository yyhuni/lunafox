# Workflow Profile Config Clone Design

**日期：** 2026-03-09

## 目标

让 `cloneWorkflowConfig` 真正与其名字语义一致：返回与 loader 内部配置对象彻底隔离的配置副本，避免调用方修改嵌套 map / slice 时污染 loader 持有的 profile。

## 设计方向

- 保持 `WorkflowProfile` 的公开结构不变
- 保持 adapter 现有读取路径不变
- 将配置 clone 从“顶层浅拷贝”收敛为“递归深拷贝”
- 仅支持当前配置模型会出现的常见 JSON/YAML 结构：`map[string]any`、`[]any` 与标量值

## 方案比较

- 方案 A：保留浅拷贝，仅改函数名为 `cloneWorkflowConfigShallow`
  - 优点：零行为变化
  - 缺点：不能解决嵌套引用共享问题
- 方案 B：通过 JSON marshal/unmarshal 做深拷贝
  - 优点：代码短
  - 缺点：多一次序列化成本，且会改变部分值类型语义
- 方案 C：递归深拷贝 `map[string]any` / `[]any`
  - 优点：语义明确，成本低，不依赖额外序列化
  - 缺点：需要手写少量递归逻辑

## 推荐方案

采用方案 C：递归深拷贝 `map[string]any` / `[]any`，其他值按值返回。

## 实现边界

- 仅修改 `server/internal/bootstrap/wiring/catalog/wiring_catalog_workflow_profile_query_store_adapter.go`
- 新增同目录单元测试验证嵌套 map / slice 不共享引用
- 不修改 loader、catalog DTO、handler API 结构

## 验证方式

- `go test ./internal/bootstrap/wiring/catalog -count=1`
