# Workflow Schema

`server/internal/workflow/schema` 现在只承担两件事：

- 嵌入并编译 `*.schema.json`
- 提供 workflow 配置的 JSON Schema 校验能力

它**不再**承担 workflow 目录元数据职责：

- 不再从 schema 解析 `workflowId / displayName / stages`
- 不再暴露 `ListWorkflowMetadata`
- 不再把 schema 当成 catalog 的事实源

## 当前职责

- `ValidateConfigMap(workflowId, config)`：校验单个 workflow 的配置对象
- `ValidateYAML(workflowId, yamlBytes)`：校验 YAML；如果存在顶层 `workflowId:` 包装则自动下钻
- `ListWorkflows()`：通过嵌入的 `*.schema.json` 文件名推导可用 workflowId

## 工件分层

- `schema`：纯 JSON Schema 校验
- `manifest`：workflow 元数据、阶段/工具声明、默认 profile 引用、默认值语义
- `profile`：独立 YAML 模板工件

## 命名约定

- schema 文件名：`<workflowId>.schema.json`
- schema `$id`：`lunafox://schemas/workflows/<workflowId>`

例如：

- 文件：`subdomain_discovery.schema.json`
- `$id`：`lunafox://schemas/workflows/subdomain_discovery`
