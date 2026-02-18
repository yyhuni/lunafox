# LunaFox Server Repository 与分层规范 v1（DDD-Strict 增强）

## 1. 目标

在不改变 HTTP API 行为与业务语义的前提下，统一 `server/internal/modules/*` 的分层边界与 repository 结构，降低维护成本并防止层间耦合回退。

## 2. 适用范围

- `server/internal/modules/*/application`
- `server/internal/modules/*/domain`
- `server/internal/modules/*/repository`
- `server/internal/modules/*/repository/persistence`
- `server/internal/modules/*/handler`

## 3. 目录终态

每个模块统一为：

- `application/`
- `domain/`
- `repository/`
- `repository/persistence/`
- `dto/`
- `handler/`
- `router/`

## 4. Repository 文件命名规范

每个资源优先采用：

- `<resource>.go`：仓储结构体、构造器、公共常量/类型
- `<resource>_query.go`：`Find/Get/List/Exists/Count/Stream/Scan/Pull`
- `<resource>_command.go`：`Create/Update/Delete/Bulk*/SoftDelete/Mark*/Save/Cancel/Fail/Unlock/Unlink/Add`
- `<resource>_adapter.go`（可选）：domain port 适配
- `<resource>_sql.go`（可选）：SQL 常量语句

## 5. 禁止项

- 禁止 `*_mutation.go`
- 禁止 repository 中使用泛名 `types.go`
- 禁止 `*_query.go` 出现写操作方法
- 禁止 `*_command.go` 出现查询方法
- 禁止存在 `modules/*/service`、`modules/*/model` 目录

## 6. 分层依赖约束（DDD-Strict）

- handler 允许依赖：application / dto / httpdto / pkg
- application 核心禁止依赖：handler / router / middleware
- domain 禁止依赖：application / repository / dto / handler / router
- repository 禁止依赖：handler / router
- handler 禁止直接依赖：`service` / `model` / `repository/persistence`
- domain 禁止出现 `gorm:"..."` 标签

## 7. DTO 共享层协同规范

- `server/internal/modules/httpdto` 是共享 HTTP DTO 唯一实现源
- `server/internal/modules/*/dto/common_http.go` 必须与模板一致
- 模板：`server/scripts/templates/common_http.go.tmpl`

## 8. 守卫与校验

- `make check-architecture`
  - `scripts/check-no-legacy-layer.sh`
  - `scripts/check-layer-dependencies.sh`
  - `scripts/check-domain-purity.sh`
  - `scripts/check-ddd-boundaries.sh`
  - `scripts/check-dto-boundaries.sh`
  - `scripts/check-repository-boundaries.sh`
  - `scripts/check-common-httpdto.sh`
- `make check-ddd`：兼容入口，等价于 `make check-architecture`
- `make check-dto-selftest`：DTO 守卫自测
- `make check-repo-selftest`：repository 守卫自测
- `make sync-common-httpdto`：同步模块 `common_http.go`

`scripts/check-layer-dependencies.sh` 中 strict 模块当前为：

- `agent`
- `security`
- `catalog`
- `identity`
- `asset`
- `scan`
- `snapshot`

## 9. 落地状态

- 已移除：`modules/*/service`、`modules/*/model`
- 已统一：`modules/*/repository/persistence`
- 已切换：原 service 调用链路收敛到 `application`
- 已保持：HTTP API、路由、状态码、JSON 字段语义不变
- 已完成：`snapshot` 模块 application 不再依赖 repository/persistence 实现层（通过 bootstrap 适配器注入）

### 9.1 snapshot 组装模式（已落地）

- 业务入口：`bootstrap/wiring.go` 统一组装 Query/Command service 后注入 Facade。
- 适配器承载：`bootstrap/wiring_snapshot_adapters.go`（scan lookup、snapshot store、asset/security sync、raw output codec）。
- application Facade 构造函数仅接收应用层对象，不直接接收 repository 或外部模块 service。

## 10. 变更原则

- 不改数据库 schema
- 不改对外 API 语义
- 优先做边界重排与守卫收口
