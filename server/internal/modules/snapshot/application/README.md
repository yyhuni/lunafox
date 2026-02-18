# snapshot/application

通用规则请遵循：`docs/server-application-naming-template-v1.md`

snapshot 模块补充规则：

- **入口聚合**：facade 维度按业务视角划分（`facade_web_snapshot.go`、`facade_discovery_snapshot.go`、`facade_port_capture_snapshot.go`、`facade_vulnerability_snapshot.go`）。
- **服务编排**：`*_snapshot.go` 按资产类型拆分 query/command 服务实现，不内联端口或跨边界模型定义。
- **端口拆分**：端口按资源+职责拆分为 `*_query_ports.go`、`*_command_ports.go`、`*_lookup_ports.go`、`*_codec_ports.go`。
- **模型命名**：跨边界模型和入参分别使用 `*_item_models.go`、`*_query_inputs.go`（如 `snapshot_list_query_inputs.go`、`vulnerability_snapshot_query_inputs.go`），domain 别名使用 `*_aliases.go`。
- **历史迁移**：无历史聚合文件，保持资源化命名不回退。
