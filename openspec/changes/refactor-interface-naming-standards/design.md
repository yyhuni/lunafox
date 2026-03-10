# Design: 边界命名标准化

## Decision
采用“按边界统一”的标准方案，而不是全仓库只允许一种大小写：

1. HTTP / JSON 对外契约统一 `camelCase`
2. Gin context key 不作为开放契约，统一封装为 accessor
3. 结构化日志统一为语义化字段名：
   - HTTP 字段优先对齐 OpenTelemetry semantic conventions
   - 领域字段使用点分命名空间（如 `agent.id`、`scan.id`、`workflow.id`）
4. 数据库列名、GORM `column:` 标记和外部工具占位符保留既有 `snake_case`
5. gRPC / HTTP 错误文案中的字段名与 API 契约保持一致，使用 `camelCase`
6. 命名型 path param 在显式暴露时使用 `camelCase`；通用资源标识继续使用 `:id`
7. Loki / Prometheus label 名称保持 `snake_case`，因为受外部 label 语法限制

## Scope
### In Scope
- `json` tag 命名统一
- request / user / agent 的 Gin context key 封装
- HTTP request / recovery 日志语义化
- 已有业务日志中的 `snake_case` 字段收敛
- 测试与项目约定同步更新

### Out of Scope
- 数据库列名、SQL、Loki label 语法
- 第三方工具要求的 `snake_case` 占位符，例如 `api_key`
- Loki / Prometheus label 之外的观测字段兼容桥接

## Naming Rules
### JSON
- `requestId`, `agentId`, `pageSize`, `updatedAt`
- 禁止新增 `request_id`, `agent_id`, `updated_at`

### Error messages
- 错误文案中提到字段时使用 `scanId`, `targetId`, `itemsJson`

### Path params
- 显式命名参数使用 `:scanId`, `:targetId`, `:agentId`, `:taskId`
- 已采用通用资源标识的 `:id` 路由不需要改名

### Loki / Prometheus labels
- 保持 `agent_id`, `container_name` 这类 `snake_case` label 名称
- 不使用点分命名，因为外部 label 语法不支持

### Gin Context
- 使用 helper：`setRequestID` / `GetRequestID`
- 使用 helper：`setUserClaims` / `GetUserClaims`
- 使用 helper：`setAgent` / `GetAgent` / `GetAgentID`
- 允许内部 key 字面值存在，但不允许业务代码跨包直接依赖这些字符串

### Structured Logs
#### HTTP / observability fields
- `request.id`
- `http.request.method`
- `http.response.status_code`
- `url.path`
- `url.query`
- `client.address`
- `user_agent.original`
- `http.response.body.size`
- `http.server.request.duration_ms`

#### Domain fields
- `agent.id`
- `agent.last_heartbeat`
- `task.id`
- `scan.id`
- `scan.status`
- `workflow.id`
- `reject.reason`
- `retry.interval`
- `loki.url`
- `server.grpc.port`

## Migration Strategy
- 不保留双写、不保留 alias、不保留 fallback。
- 所有测试先改为表达新规范，再最小实现通过。
- 优先改 HTTP / middleware / logger 基础设施，再改业务日志与内部 JSON tag。

## Enforcement
- 新增 `scripts/ci/check-interface-naming.sh` 作为仓库级守门脚本，使用 `rg` 扫描高价值规则。
- 脚本检查以下边界回流：
  - `json` tag 中新增 `snake_case`
  - `zap` 字段 key 使用裸 `snake_case`
  - 业务代码跨包直接读取 `requestId` / `userClaims` / `agentId` / `agent` 这类 Gin context key
  - 中间件 / gRPC 边界重新引入 `request_id`、`scan_id`、`target_id`、`items_json`
  - 显式命名路由参数重新引入 `:scan_id`、`:target_id`
- 脚本 allowlist 与 `docs/plans/2026-03-06-interface-naming-audit.md` 保持一致，明确放行数据库列名、SQL、Loki / Prometheus labels、第三方 provider 占位符、legacy 反例测试和 generated 文件。
- GitHub Actions 通过现有 `ci.yml` 的 path filter 只在相关路径变更时运行该检查。

## Verification
- `go test ./server/internal/middleware ./server/internal/pkg`
- `go test ./server/internal/job ./server/internal/modules/scan/application`
- `go test ./server/internal/cache ./server/internal/modules/catalog/repository ./server/internal/modules/agent/repository`
- `go test ./server/internal/grpc/runtime/service ./server/internal/modules/agent/application ./server/internal/modules/snapshot/handler`
- `openspec validate refactor-interface-naming-standards --strict --no-interactive`
