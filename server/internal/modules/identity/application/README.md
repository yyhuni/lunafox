# identity/application

通用规则请遵循：`docs/server-application-naming-template-v1.md`

identity 模块补充规则：

- **入口聚合**：facade 按 auth/user/organization 领域能力拆分（`facade_auth.go`、`facade_user.go`、`facade_organization.go`）。
- **服务编排**：user 与 organization 按读写职责拆分 `*_query.go`、`*_command.go`；auth 维持独立认证流程编排。
- **端口拆分**：auth 依赖拆分为 `auth_ports.go`、`auth_user_query_ports.go`；user/organization 使用对应 `*_query_ports.go`、`*_command_ports.go`。
- **模型命名**：新增跨边界模型优先资源化命名（如 `*_item_models.go`、`*_query_inputs.go`），避免泛名文件。
- **历史迁移**：`aliases.go`、`errors.go` 为历史聚合文件，后续新增优先资源化命名。
