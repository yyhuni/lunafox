# Change: HTTP 头命名现代化

## Why
当前仓库仍混用多类 `X-*` HTTP 头：
- 应用观测头 `X-Request-ID`
- 应用私有认证头 `X-Agent-Key`、`X-Worker-Token`
- 代理兼容头 `X-Forwarded-*`、`X-Real-IP`

IETF 已不推荐为新协议元素继续使用 `X-` 前缀，但仓库里这几类头的性质并不相同。若不加区分地“一次性全部去 `X-`”，会误把代理兼容层和应用协议层混在一起；若完全不动，则会继续积累应用层命名债务。

当前项目仍处于开发期，适合一次性收敛项目自定义的应用头命名，并把基础设施兼容头明确排除在迁移范围外。

## What Changes
- 将请求关联头从 `X-Request-ID` 收敛为 `Request-Id`，不保留兼容桥接。
- 将 agent HTTP 认证从 `X-Agent-Key` 收敛为 `Authorization: Bearer <agent-token>`。
- 将 `X-Worker-Token` 视为遗留 HTTP 认证方案；新变更不得继续依赖该头，现有中间件需在实现阶段做去留确认或隔离。
- 明确 `X-Forwarded-*`、`X-Real-IP` 属于代理兼容层，继续保留。
- 为受影响中间件、测试和文档补充回归，确保后续新增边界不再引入新的应用层 `X-*` 头。

## Impact
- Affected specs:
  - `http-header-contract`
  - `runtime-auth`
- Affected code:
  - `server/internal/middleware/logger.go`
  - `server/internal/middleware/logger_test.go`
  - `server/internal/middleware/agent_auth.go`
  - `server/internal/middleware/agent_auth_test.go`
  - `server/internal/middleware/worker_auth.go`
  - `docker/nginx/nginx.conf`
  - request examples / docs referencing old headers
