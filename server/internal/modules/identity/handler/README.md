# identity/handler

identity 模块 handler 规范：

- 按资源拆分 handler 文件（auth/user/organization）。
- 主资源 canonical collection list handler 使用 `List`，不要在 receiver 已表达资源时重复写 `ListUsers` / `ListOrganizations`。
- organization 主资源按读写职责拆分：
  - `organization_query.go`：查询接口（list/get）
  - `organization_command.go`：写接口（create/update/delete/batch-delete）
- organization 关联目标按读写职责拆分：
  - `organization_targets_query.go`：查询接口（list targets）
  - `organization_targets_command.go`：写接口（link/unlink targets）

约束：

- handler 层仅处理 HTTP 绑定、错误映射与响应组装，不承载业务编排。
- 业务读写分层通过 application/facade 的 query/command 服务实现。
- 新增或迁移 HTTP 边界时不要复制 legacy hyphenated action 路径风格；批量写入应按 `openspec/changes/standardize-project-boundary-contracts/` 收敛到 AIP batch method 或登记为迁移例外。
