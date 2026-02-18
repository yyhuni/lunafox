# catalog/application

通用规则请遵循：`docs/server-application-naming-template-v1.md`

catalog 模块补充规则：

- **入口聚合**：facade 按 target/engine/wordlist 业务视角拆分（含 `facade_target_batch.go`、`facade_target_detail.go`、`facade_wordlist_content.go`、`facade_wordlist_create.go`）。
- **服务编排**：target command 按场景拆分为 `target_command_crud.go` 与 `target_command_batch.go`；其余资源遵循 query/command 分层。
- **端口拆分**：端口按资源职责拆分，wordlist 文件能力采用 port + default implementation（`wordlist_file_ports.go` + `local_wordlist_file_store.go`）。
- **实现唯一性**：`WordlistFileStore` 的默认实现仅保留在 `application/local_wordlist_file_store.go`，避免在 `infrastructure` 层出现同名重复实现造成歧义。
- **模型命名**：新增输入/输出/中间模型优先资源化命名（如 `*_query_inputs.go`、`*_item_models.go`）。
- **历史迁移**：`aliases.go`、`errors.go` 为历史聚合文件，后续新增优先资源化命名。
