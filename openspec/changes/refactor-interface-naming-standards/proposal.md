# Change: 边界命名标准化

## Why
当前仓库在对外 JSON、Gin context key、错误响应、结构化日志字段之间混用了多套命名风格：
- API JSON 大多使用 `camelCase`，但仍存在少量 `snake_case` `json` tag。
- Gin context key 以裸字符串分散在中间件中，既不统一也不易约束。
- HTTP 请求日志已经开始从 `camelCase` 迁到 `snake_case`，但这并不是跨边界的最终标准。
- 业务日志字段仍以历史 `snake_case` 为主，和 HTTP 语义字段没有统一的命名体系。

在不保留兼容桥接的前提下，需要一次性把“边界命名”收敛到更标准的方案：
- HTTP / JSON 契约使用 `camelCase`
- Gin context key 只通过 accessor 暴露
- 结构化日志使用语义化命名；HTTP 相关字段优先对齐 OpenTelemetry semantic conventions

## What Changes
- 将 API / 错误响应 / 内部 JSON 序列化使用的 `snake_case` `json` tag` 收敛为 `camelCase`，除数据库列名和外部工具占位符外不再保留历史命名。
- 为 request/user/agent 中间件上下文引入统一 accessor，消除业务代码对裸 `c.Set` / `c.Get` 字符串 key 的直接依赖。
- 将 HTTP request / recovery 日志改为语义化字段名；现有业务日志字段从 `snake_case` 迁移到点分语义命名。
- 将 gRPC 对外错误文案中的字段名统一为 `camelCase`。
- 将测试与显式命名的 path param 风格统一为 `camelCase`；保留通用 `:id` 路由不变。
- 明确 Loki / Prometheus 外部 label 名称保持 `snake_case`，因为受外部语法约束。
- 更新项目约定、设计文档和回归测试，确保后续新增代码沿用同一命名规则。
- 新增仓库级命名检查脚本并接入 GitHub Actions，作为边界命名的自动化守门。

## Impact
- Affected specs: `boundary-naming`
- Affected code:
  - `server/internal/middleware/*`
  - `server/internal/pkg/logger*`
  - `server/internal/cache/*`
  - `server/internal/modules/**/repository/persistence/*`
  - `server/internal/modules/**/application/*log*`
  - `openspec/project.md`
  - `docs/plans/*interface-naming-standards*`
  - `docs/plans/*interface-naming-enforcement*`
  - `scripts/ci/check-interface-naming.sh`
  - `.github/workflows/ci.yml`
