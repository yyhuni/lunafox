# identity/application

通用规则请遵循：`server/docs/server-application-naming-template-v1.md`

identity 模块补充规则：

- **入口聚合**：facade 按 auth/user/organization 领域能力拆分（`facade_auth.go`、`facade_user.go`、`facade_organization.go`）。
- **Facade 边界**：facade 只保留入口委派、轻量投影和边界错误语义映射调用；重复的 user/organization not-found 映射统一收口到 `identity_facade_helpers.go`。
- **服务编排**：user 与 organization 按读写职责拆分 `*_query.go`、`*_command.go`；auth 维持独立认证流程编排。
- **adapter 例外**：当前无 transport-facing 或 cross-module bridge 型 adapter service 例外。
- **端口拆分**：auth 依赖拆分为 `auth_ports.go`、`auth_user_query_ports.go`；user/organization 使用对应 `*_query_ports.go`、`*_command_ports.go`。
- **模型命名**：新增跨边界模型优先资源化命名（如 `*_item_models.go`、`*_query_inputs.go`），避免泛名文件。
- **历史迁移**：`aliases.go`、`errors.go` 已完成首批迁移，继续保持资源化命名不回退。
- **Organization 语义**：organization 是 target 的全局业务分组，不是用户成员关系、租户或 MCP 数据范围。创建命令的 context-aware 路径必须保留 caller cancellation；active 名称由数据库 partial unique index 最终保证，soft-deleted 名称可复用，重复创建统一返回 `ErrOrganizationExists` 而不暴露持久化诊断。
