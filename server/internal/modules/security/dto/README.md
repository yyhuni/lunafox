# security/dto

security 模块 DTO 规范：

- `*_dto.go`：按资源拆分放置漏洞与安全相关 DTO。
- `httpdto_adapter.go`：仅作为薄适配层，重导出 `server/internal/modules/httpdto` 的共享 HTTP DTO 能力（绑定、分页、统一错误响应）。

约束：

- security DTO 独立维护，不复用 asset/snapshot 的业务 DTO。
- 快照同步由 `snapshot/service` 做显式转换，不在 DTO 层耦合。
