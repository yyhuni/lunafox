# snapshot/dto

snapshot 模块 DTO 规范：

- `*_snapshot_dto.go`：按资源拆分放置快照相关 DTO（endpoint/website/directory/host-port/screenshot/subdomain/vulnerability）。
- `httpdto_adapter.go`：仅作为薄适配层，重导出 `server/internal/modules/httpdto` 的共享 HTTP DTO 能力（绑定、分页、统一错误响应）。

关键约束：

- `snapshot/dto` 禁止 import `asset/dto` 与 `security/dto`。
- 不允许通过类型别名复用 asset/security DTO。
- 与资产/漏洞的字段映射必须在 `snapshot/service` 的转换层完成。
