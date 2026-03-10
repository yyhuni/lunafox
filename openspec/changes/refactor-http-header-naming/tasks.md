## 1. OpenSpec and design sync
- [x] 1.1 审阅并确认 `proposal.md`、`design.md`、delta specs 的最终边界
- [x] 1.2 确认 `docs/plans/2026-03-06-http-header-naming-design.md` 与 OpenSpec 设计保持一致

## 2. Request ID header migration (TDD)
- [x] 2.1 在 `server/internal/middleware/logger_test.go` 增加基于 `Request-Id` 的失败测试（RED）
- [x] 2.2 运行 `go test ./server/internal/middleware -run TestLogger`，确认旧实现不满足新头契约
- [x] 2.3 将 `server/internal/middleware/logger.go` 中请求关联头常量切换为 `Request-Id`（GREEN）
- [x] 2.4 运行 `go test ./server/internal/middleware -run TestLogger`，确认相关测试通过

## 3. Agent auth header migration (TDD)
- [x] 3.1 在 `server/internal/middleware/agent_auth_test.go` 增加 `Authorization: Bearer <agent-token>` 的失败测试（RED）
- [x] 3.2 运行 `go test ./server/internal/middleware -run TestAgentAuth`，确认旧实现不满足新认证契约
- [x] 3.3 将 `server/internal/middleware/agent_auth.go` 收敛为解析 `Authorization` 头（GREEN）
- [x] 3.4 更新 agent 请求示例与注释，移除 `X-Agent-Key`
- [x] 3.5 运行 `go test ./server/internal/middleware -run TestAgentAuth`，确认相关测试通过

## 4. Worker auth legacy audit
- [x] 4.1 审计 `server/internal/middleware/worker_auth.go` 的生产路由与引用关系
- [x] 4.2 若无生产依赖，则删除或隔离 legacy `WorkerAuthMiddleware`
- [x] 4.3 若需短期保留，则明确标记 legacy 并禁止新路由继续接入 `X-Worker-Token`
- [x] 4.4 运行受影响测试或回归检查，确认无遗留引用

## 5. Verification
- [x] 5.1 全仓搜索 `X-Request-ID`、`X-Agent-Key`、`X-Worker-Token`，更新测试、注释和示例请求
- [x] 5.2 确认 `docker/nginx/nginx.conf` 中 `X-Forwarded-*`、`X-Real-IP` 未被误改
- [x] 5.3 运行 `go test ./server/internal/middleware ./server/internal/bootstrap/...`
- [x] 5.4 运行 `openspec validate refactor-http-header-naming --strict --no-interactive`
