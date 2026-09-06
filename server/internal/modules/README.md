# modules 命名约定

适用范围：`server/internal/modules/*`

## domain 目录文件命名

- 优先使用具体领域概念命名，如 `scan.go`、`task.go`、`vulnerability.go`、`workflow_catalog.go`。
- 当文件承载模块内唯一主实体集合，且没有更强的业务名可用时，才使用 `entity.go`。
- 当一个文件暂时承载多个并列实体时，可使用 `entities.go`；但这不是默认选项，后续继续增长时应优先按资源或概念拆分。
- 领域层横切语义继续使用固定职责名，如 `rules.go`、`events.go`、`repository.go`、`errors.go`。
- 避免使用语义过弱的泛名文件，如 `types.go`、`common.go`、`models.go`、`utils.go`。
- 不为统一而统一。存量文件保持兼容；仅在模块发生实质改动且重命名能明显提升可读性时，再顺手收口。

## application 目录文件命名

- `ports.go`：应用层端口接口定义（依赖仓储、外部能力、时间/随机等抽象）。
- `<module>_facade.go`：对外聚合入口，组合 query/command/service。
- `<module>_<role>_service.go`：按职责拆分的应用服务实现。
- `errors.go`：应用层错误定义。
- `codec.go` / `mapper.go`：仅在应用层确有转换职责时使用能力命名文件。

## infrastructure 目录文件命名

- `clock.go` / `token_generator.go` / `codec.go`：按能力拆分默认实现文件（推荐）。
- 默认实现优先放在 `infrastructure`，并通过 wiring 注入到 application。

## HTTP DTO 资源名字段

- AIP-governed response DTO 中，`json:"name"` 表示 canonical resource name，例如 `scans/{scan}`、`targets/{target}`、`wordlists/{wordlist}`。
- 给用户展示的业务名称不要复用 response `name`；使用 `displayName`，或使用更具体的业务字段名。
- Go 字段名可保留 `Name` 以贴合 JSON contract；若局部代码需要消除歧义，可命名为 `ResourceName` 并继续使用 `json:"name"`，但不得改变对外 JSON 字段语义。
- create/update request DTO 中的 `json:"name"` 可以是用户提交的业务名或自然键，例如 organization/target/wordlist 的创建名称；不要把 response `name` 规则机械套到 request 输入上。
- persistence/domain 层的 `Name` 可能仍表示业务名、自然键或数据库列；跨到 handler/dto 时必须显式映射，不能假设各层 `Name` 同义。
- resource name 字符串统一通过 `server/internal/modules/httpdto/resource_name.go` 生成；handler 不要手写 `scans/%d`、`targets/%d` 等格式化逻辑。
- 路由参数名可以使用资源语义（如 `:scan`、`:target`、`:organization`），但参数值形态以当前 route contract 为准；迁移期 handler 里仍可能解析裸 ID，不要因为参数名是资源名就擅自改为解析 `scans/{scan}`。
- Gin 不能同时注册多个同前缀的 Google-style custom method 路由（如 `:batchCreate` 和 `:batchDelete`）；需要用单个 wildcard route 进入 `httpdto.DispatchCustomMethod`。该 helper 只允许枚举 canonical verb，不能作为 legacy alias 或兼容分发层使用。

## HTTP 路由注释与挂载前缀

- 后端 first-party HTTP application routes 统一由 bootstrap 挂载在 `/v1` 下；模块 router 只注册资源相对路径，例如 `/wordlists`、`/targets/:target/subdomains`，不要在模块 router 内硬编码 `/v1` 或 `/api`。
- handler 上方的路由示例注释写最终对外路径，例如 `// GET /v1/wordlists`；不要写模块相对路径，也不要写旧 `/api` 前缀。
- 路由示例注释必须匹配 Gin 实际注册路径；实际 route 没有尾斜杠时，注释也不要补尾斜杠。
- 在新增或评审 router path、handler 路由注释、proto RPC、runtime payload 名称前，先判定该边界是稳定资源、fixed alias view、custom method、long-running operation/task，还是 protocol-native operational/media contract；不要机械把注册、续期、安装、导出、下载或执行动作铸造成 `*Registrations`、`*Renewals`、`*InstallScripts`、`*Statistics`、`*/exportFiles/current` 这类资源形状，除非 OpenSpec 或近代码契约已经明确证明该形状成立。

## 约束

- 新代码不再使用 `contracts.go`，统一使用 `ports.go`。
- 新代码不再使用 `defaults.go`。
- 新代码不在 `application` 层使用 `default_impls.go`。
- 默认实现文件使用能力命名，不使用泛名聚合文件。
- 避免使用语义过弱的泛名文件（如 `types.go`、`common.go`）。

## 目的

- 统一术语（Port/Adapter 语义一致）。
- 降低跨模块迁移和全局检索成本。
- 避免同类职责在不同模块出现多套命名。
