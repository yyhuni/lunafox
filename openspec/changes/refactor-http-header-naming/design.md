# Design: HTTP 头命名现代化

## Decision
采用“只治理应用层私有头，保留基础设施兼容头”的方案：

1. `X-Request-ID` 迁移为 `Request-Id`
2. `X-Agent-Key` 迁移为 `Authorization: Bearer <agent-token>`
3. `X-Worker-Token` 不再作为新协议入口继续扩展，先按遗留 HTTP 认证方案处理
4. `X-Forwarded-*`、`X-Real-IP` 保留

该方案在开发期可以一次性清理项目自定义协议命名，同时避免把代理层事实标准错误纳入重构范围。

## Scope
### In Scope
- `server/internal/middleware/logger.go` 的请求头命名
- `server/internal/middleware/agent_auth.go` 的 agent HTTP 认证输入
- `server/internal/middleware/worker_auth.go` 的遗留状态确认
- 中间件测试、请求示例和注释同步更新

### Out of Scope
- `docker/nginx/nginx.conf` 中代理兼容头的协议升级
- W3C `traceparent` 的完整接入
- 需要同一请求同时承载用户 JWT 与 agent / worker 凭证的双凭证通道设计

## Contracts
### Request correlation
- 应用 MUST 读取并写回 `Request-Id`
- 应用 MUST NOT 继续把 `X-Request-ID` 作为规范头名
- 结构化日志字段保持现有语义化写法，不因头名变化回退到旧命名

### Agent authentication
- agent HTTP 认证 MUST 使用 `Authorization: Bearer <agent-token>`
- `/api/agent/*` 路由组与用户 JWT 路由保持独立认证边界
- 在没有双凭证同请求需求前，MUST NOT 新增第二个应用层私有认证头

### Worker authentication
- 当前 worker runtime 数据写入不再经由 HTTP `/api/worker/*`
- `WorkerAuthMiddleware` MUST 被视为 legacy code，不能作为新 HTTP 协议演进基础
- 若未来恢复 HTTP worker 路由，新认证 MUST 优先使用 `Authorization`

### Proxy compatibility
- 代理兼容头继续保留在基础设施层
- 本次变更 MUST NOT 因“去 `X-`”目标而改写 `X-Forwarded-*` 或 `X-Real-IP`

## Migration Strategy
- 不做双写，不做旧头 fallback。
- 先改测试表达新契约，再做最小实现。
- 先处理 `logger` 与 `agent_auth` 两个真实入口，再处理 `worker_auth` 的遗留清理。
- 全仓搜索旧头名，保证测试、示例请求、文档和注释同步收敛。

## Risks
- 测试或文档仍残留旧头名，导致后续误用
- 未来 agent 可能出现双凭证同请求场景，迫使再引入第二认证通道
- 对 `worker_auth` 的真实产品地位判断失误，导致过早删除仍需保留的遗留代码

## Mitigations
- 在实现前先审计所有旧头名引用并纳入任务清单
- 把 agent 认证变更限制在独立路由边界，不与用户 JWT 路由混用
- 对 `worker_auth` 先做路由与引用审计，再决定删除还是 legacy 隔离
