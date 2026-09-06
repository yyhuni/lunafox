# asset/dto

asset 模块 DTO 规范：

- `*_dto.go`：按资源拆分放置 asset 业务 DTO（subdomain/website/endpoint/directory/host-port/screenshot）。
- `batch_delete_dto.go`：放置跨 2 个及以上资源复用的业务 DTO（当前为批量删除请求/响应）。
- `httpdto_adapter.go`：仅作为薄适配层，重导出 `server/internal/modules/httpdto` 的共享 HTTP DTO 能力（绑定、分页、统一错误响应）。

约束：

- 不得在 DTO 文件（`dto.go` 或 `*_dto.go`）中复用 `server/internal/dto` 的业务 DTO 别名。
- 不得 import 其他模块的 `dto`（如 `security/snapshot`）。
- 跨模块数据转换放在 service/application 层，不放在 DTO 层。
- 仅当业务 DTO 在同模块内跨 2 个及以上资源复用时，才新增类似 `batch_delete_dto.go` 的通用业务 DTO 文件。
- Target Directory read DTO 的 `contentLength` 与纳秒 `duration` 只能输出精确十进制 `string` 或历史未知值 `null`；零值必须是 `"0"`，不得输出 JSON number 或混合 wire type。

## Batch 操作限制

所有批量操作的数组字段统一上限 **5,000**（`max=5000`），适用于 batch create、batch delete、batch upsert。

全局搜索的 query DTO 使用 `q`、`assetType`、`pageSize`、`pageToken` camelCase 字段，
响应只包含 `results` 和可选 `nextPageToken`。搜索结果直接复用 Website/Endpoint
read DTO，不在 DTO 层拼接技术栈、转换响应头或推断漏洞归属。
