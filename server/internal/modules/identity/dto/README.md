# identity/dto

identity 模块 DTO 规范：

- `*_dto.go`：按资源拆分放置 identity 业务 DTO（organization/user/target）。
- `batch_delete_dto.go`：放置跨 2 个及以上资源复用的业务 DTO（当前为批量删除请求/响应）。
- `httpdto_adapter.go`：仅作为薄适配层，重导出 `server/internal/modules/httpdto` 的共享 HTTP DTO 能力（绑定、分页、统一错误响应）。

约束：

- 不得在 DTO 文件（`dto.go` 或 `*_dto.go`）中复用 `server/internal/dto` 的业务 DTO 别名。
- 不得 import 其他业务模块的 `dto`。
- 仅当业务 DTO 在同模块内跨 2 个及以上资源复用时，才新增类似 `batch_delete_dto.go` 的通用业务 DTO 文件。

## Batch 操作限制

所有批量操作的数组字段统一上限 **5,000**（`max=5000`），适用于 batch delete、batch link/unlink。
