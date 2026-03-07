# Interface Naming Standards Design

## 背景
当前仓库在 API JSON、Gin context、错误响应、结构化日志之间使用了多套命名规则，导致 review 反复讨论、查询口径不统一、迁移成本越来越高。

## 目标
- 对外 JSON 统一 `camelCase`
- Gin context 统一走 accessor
- 结构化日志统一走语义化字段名
- 不保留兼容桥接

## 方案
### 1. JSON
所有 `json` tag 默认统一为 `camelCase`。数据库列名、GORM `column:` 标记、URL path 参数和第三方工具占位符不在本次变更范围。

### 2. Gin context
保留 `gin.Context` 的使用方式，但把 request / user / agent 相关数据写入和读取全部封装到 `middleware` helper 中，避免业务层依赖裸字符串 key。

### 3. Logs
HTTP 相关日志字段优先对齐 OTel 语义：
- `request.id`
- `http.request.method`
- `http.response.status_code`
- `url.path`
- `url.query`
- `client.address`
- `user_agent.original`
- `http.response.body.size`

领域日志统一使用点分命名空间：
- `agent.id`
- `scan.id`
- `task.id`
- `workflow.id`
- `scan.status`

## 验证
先改测试表达新规则，再最小实现；最后跑 middleware / pkg / job / scan application / cache / repository 相关测试与 `openspec validate`。

## 4. Loki labels and path params
- Loki / Prometheus label 名称不是内部日志字段，继续保持 `agent_id`、`container_name` 这类 provider-compatible `snake_case`。
- 显式命名的 path param 与错误文案统一使用 `camelCase`，例如 `scanId`、`targetId`、`itemsJson`。
- 已经采用通用资源标识的 `:id` 路由不需要为了形式统一而重写。
