# scan/dto

scan 模块 DTO 规范：

- `*_dto.go`：按资源拆分放置 scan/scan-log/task 相关 DTO。
- `httpdto_adapter.go`：仅作为薄适配层，重导出 `server/internal/modules/httpdto` 的共享 HTTP DTO 能力（绑定、分页、统一错误响应）。

约束：

- scan 业务 DTO 独立维护，不复用共享层业务 DTO（DTO 文件可为 `dto.go` 或 `*_dto.go`）。
- 仅保留对共享 HTTP DTO 的依赖。
