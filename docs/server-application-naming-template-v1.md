# Server Application Naming Template v1

适用范围：`server/internal/modules/*/application`

## 核心命名规则

- 端口接口文件使用资源化命名：
  - `*_query_ports.go`
  - `*_command_ports.go`
  - `*_lookup_ports.go`
  - `*_codec_ports.go`
- 应用错误文件：`*_errors.go`
- 跨边界输入/中间模型：`*_item_models.go`
- domain 类型别名：`*_aliases.go`
- 应用层编解码实现：`*_codec.go`
- 对外编排入口：`facade_*.go`
- 可选服务拆分：`*_service.go`（仅在 facade 无法承载复杂编排时启用）

## 设计约束

- 新代码不使用 `contracts.go`。
- 禁止新增泛名 `ports.go`、`errors.go`。
- 新代码不使用 `defaults.go`。
- 新代码不在 `application` 层新增 `default_impls.go`。
- `application` 层不直接依赖 `dto`；DTO 映射放在 `handler` / `wiring`。
- 避免弱语义泛名（如 `types.go`、`common.go`）。
- 默认实现若属于外部能力适配，优先放在 `infrastructure`，并按能力命名。

## 模块补充规则字段模板（固定 5 条）

- **入口聚合**：说明 facade 或对外入口如何收敛。
- **服务编排**：说明应用服务拆分方式与主流程文件。
- **端口拆分**：说明端口按资源/职责如何拆分。
- **模型命名**：说明输入/输出/item/alias 等模型命名策略。
- **历史迁移**：说明历史聚合文件状态与后续迁移方向；若无历史文件，明确写“无历史聚合文件”。

## 使用方式

- 每个模块在 `application/README.md` 中引用本模板。
- 模块 README 只保留“模块特有补充规则”，避免重复复制通用规则。
- 模块补充规则固定使用以上 5 条字段，保持跨模块可对比性。
