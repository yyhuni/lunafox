# workflow/schema

该目录负责读取内嵌的 `*.schema.json`，并提供：

- workflow 配置校验能力（`ValidateConfigMap` / `ValidateYAML`）
- workflow 元数据枚举能力（`ListWorkflows` / `ListWorkflowMetadata`）

## 元数据约束

每个 schema 文件必须声明：

- `x-workflow`：workflow 唯一标识（程序标识）
- `x-metadata.name`：workflow 展示名（人类可读）

可选字段：

- `description`：workflow 描述

### DisplayName 规则

- `DisplayName` 仅来自 `x-metadata.name`
- 不再回退 `title`
- 不再回退 `x-workflow`
- 如果缺少 `x-metadata.name`，`ListWorkflowMetadata` 会返回错误

这保证 catalog 接口中的 `displayName` 有单一来源，避免同一 workflow 在不同入口出现命名不一致。

## 示例

```json
{
  "x-workflow": "subdomain_discovery",
  "description": "Discover subdomains",
  "x-metadata": {
    "name": "Subdomain Discovery"
  }
}
```

