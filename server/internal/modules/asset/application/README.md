# asset/application

通用规则请遵循：`docs/server-application-naming-template-v1.md`

asset 模块补充规则：

- **入口聚合**：按资产类型拆分 facade 入口（`facade_website.go`、`facade_subdomain.go`、`facade_endpoint.go`、`facade_directory.go`、`facade_host_port.go`、`facade_screenshot.go`）。
- **服务编排**：读写流程按资源拆分为 `*_query.go` 与 `*_command.go`。
- **端口拆分**：端口按资源+职责拆分为 `*_query_ports.go`、`*_command_ports.go`；跨资源依赖单独建模（如 `target_lookup_ports.go`）。
- **模型命名**：新增跨边界模型优先使用资源化 `*_item_models.go`、`*_query_inputs.go`；避免新增泛名模型文件。
- **历史迁移**：`aliases.go`、`errors.go` 为历史聚合文件，后续新增优先资源化命名。
