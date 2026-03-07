# Interface Naming Audit

## Scope
本清单只审计“命名风格是否仍保留 `snake_case`”这件事，并区分：
- 需要继续保留的 `snake_case`
- 已经清空的 `snake_case`
- 不在本次边界命名收敛范围内的内容

不把数据库真实列名、SQL 片段、外部 label 语法误判为“还没改干净的 API 命名问题”。

## Summary
当前仓库的结论如下：
- HTTP / JSON 契约：应使用 `camelCase`
- Gin context：应通过 accessor 使用，不暴露裸字符串约定
- 结构化日志字段：应使用语义化字段名
- gRPC / HTTP 错误文案中的字段名：应使用 `camelCase`
- 显式命名 path param：应使用 `camelCase`
- 以下场景继续保留 `snake_case`，且属于**正确保留**

## Intentionally Retained Snake Case

### 1. Database schema / GORM column tags / raw SQL
这些名称是数据库物理模式的一部分，不属于 HTTP / JSON 字段命名。

代表位置：
- `server/cmd/server/migrations/000001_init_schema.up.sql`
- `server/internal/modules/scan/repository/persistence/scan.go`
- `server/internal/modules/scan/repository/persistence/task.go`
- `server/internal/modules/scan/repository/scan_task_sql.go`
- `server/internal/modules/agent/repository/agent_query.go`

保留原因：
- 与现有 PostgreSQL schema、索引、迁移、SQL 查询直接绑定
- 属于关系数据库命名，不应按 API 命名风格重写

### 2. Loki / Prometheus compatible labels
这些名称受外部 label 语法与既有查询兼容约束，继续保留 `snake_case`。

代表位置：
- `server/internal/modules/agent/application/loki_log_query_service.go`
- `server/internal/loki/client_test.go`
- `server/internal/modules/agent/install/templates/agent_install.sh`
- `server/internal/modules/agent/handler/agent_registration_handler_test.go`

当前保留项：
- `agent_id`
- `container_name`

保留原因：
- LogQL label selector 与 Docker Loki driver 外部标签约定要求稳定的 label 名称
- 点分语义字段适用于内部结构化日志，不适用于 Loki label key

### 3. Third-party tool / provider placeholders
这些名称不是 LunaFox API 字段，而是第三方工具配置格式的一部分。

代表位置：
- `server/internal/modules/catalog/repository/persistence/subfinder_provider_settings.go`
- `server/cmd/server/migrations/000001_init_schema.up.sql`

当前保留项：
- `api_key`
- `api_id`
- `api_secret`

保留原因：
- 与 subfinder provider 配置格式、历史存量数据和模板拼接格式绑定
- 改成 `camelCase` 会破坏外部工具语义

### 4. Negative assertions in tests
测试里继续出现旧的 `snake_case`，是为了断言“旧字段已经被移除”，不是规范倒退。

代表位置：
- `server/internal/middleware/logger_test.go`
- `server/internal/job/agent_monitor_test.go`
- `server/internal/modules/scan/application/log_fields_test.go`
- `server/internal/pkg/logger_test.go`

保留原因：
- 这是回归测试需要的反例断言
- 这些字符串不属于运行时对外契约

### 5. Domain values, not field names
少量 `snake_case` 是“值约定”而不是“字段命名”，例如 workflow ID 一类机器标识值。

代表位置：
- `openspec/changes/refactor-workflow-id-semantics/design.md`
- `openspec/changes/refactor-workflow-id-semantics/proposal.md`

保留原因：
- `workflowId` 这个字段本身使用 `camelCase`
- 但它承载的值按规范仍然是小写 `snake_case`

## Cleared Areas
以下区域已清空“不该保留的 `snake_case` 命名”：

### 1. HTTP / JSON fields
- 内部序列化用到的 `json` tag 已收敛为 `camelCase`
- 例如 `updatedAt`、`agentId`、`workflowConfigYAML`

### 2. Structured log fields
- 已从 `request_id` / `agent_id` / `scan_id` 等历史字段迁移为语义化字段名
- HTTP 语义优先对齐 OTel，业务语义改为点分命名空间

### 3. Explicitly named path params
- `server/internal` 下生产与测试代码里，显式命名 path param 已无剩余 `:scan_id` 风格
- 当前约定是显式命名参数用 `camelCase`，通用资源路由继续使用 `:id`

### 4. Worker module residual scan
- `worker` 源码（排除 generated）未发现本轮边界命名规则下仍需处理的残留项

## Quick Review Checklist
后续 review 时，看到 `snake_case` 可以按下面顺序判断：
1. 是 HTTP / JSON / 错误文案字段吗？如果是，应改成 `camelCase`
2. 是结构化日志字段吗？如果是，应改成语义化字段名
3. 是 Gin context key 的裸字符串依赖吗？如果是，应改成 accessor
4. 是数据库列名 / SQL / GORM `column:` 吗？如果是，保留
5. 是 Loki / Prometheus labels 吗？如果是，保留
6. 是第三方工具占位符吗？如果是，保留
7. 是测试里的 legacy 断言吗？如果是，保留

## Audit Snapshot
- `worker` 源码残留扫描：未发现适用的边界命名问题
- 生产 URL 显式命名 path param：未发现剩余 `snake_case` 风格
- 仍大量存在的 `snake_case` 主要集中在：数据库 schema / SQL、外部 labels、第三方占位符、回归测试反例
