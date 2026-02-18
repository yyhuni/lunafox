# modules 命名约定

适用范围：`server/internal/modules/*`

## application 目录文件命名

- `ports.go`：应用层端口接口定义（依赖仓储、外部能力、时间/随机等抽象）。
- `<module>_facade.go`：对外聚合入口，组合 query/command/service。
- `<module>_<role>_service.go`：按职责拆分的应用服务实现。
- `errors.go`：应用层错误定义。
- `codec.go` / `mapper.go`：仅在应用层确有转换职责时使用能力命名文件。

## infrastructure 目录文件命名

- `clock.go` / `token_generator.go` / `codec.go`：按能力拆分默认实现文件（推荐）。
- 默认实现优先放在 `infrastructure`，并通过 wiring 注入到 application。

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
