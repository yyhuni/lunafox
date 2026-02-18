# security/application

通用规则请遵循：`docs/server-application-naming-template-v1.md`

security 模块补充规则：

- **入口聚合**：对外用例入口单点收敛在 `facade_vulnerability.go`。
- **服务编排**：当前编排集中在 facade，只有当复杂度提升时才新增 `*_service.go`。
- **端口拆分**：端口统一按 vulnerability 资源命名，`vulnerability_raw_output_codec_ports.go` 与 `vulnerability_raw_output_codec.go` 必须成对维护（port + default implementation）。
- **模型命名**：`application` 文件统一 `vulnerability_*` 前缀，跨边界模型使用 `vulnerability_item_models.go`，类型别名使用 `vulnerability_aliases.go`。
- **历史迁移**：无历史聚合文件，保持资源化命名不回退。
