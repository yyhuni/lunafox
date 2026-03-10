## 1. OpenSpec and docs
- [x] 1.1 写入 proposal / design / delta spec
- [x] 1.2 写入 `docs/plans/2026-03-06-interface-naming-standards-design.md`
- [x] 1.3 写入 `docs/plans/2026-03-06-interface-naming-standards.md`

## 2. Middleware context and HTTP contract (TDD)
- [x] 2.1 先补 request / recovery / agent auth 的失败测试
- [x] 2.2 引入 request / user / agent context accessor
- [x] 2.3 收敛错误响应与中间件上下文字段命名
- [x] 2.4 运行相关中间件测试

## 3. Structured logs standardization (TDD)
- [x] 3.1 先补 logger / lifecycle / runtime / monitor 失败测试
- [x] 3.2 将 HTTP 日志字段迁移为语义命名
- [x] 3.3 将领域日志字段迁移为点分命名空间
- [x] 3.4 运行相关日志测试

## 4. JSON tag standardization (TDD)
- [x] 4.1 先补内部 JSON 序列化失败测试
- [x] 4.2 将内部 `snake_case` `json` tag 收敛到 `camelCase`
- [x] 4.3 排除数据库列名与外部工具占位符
- [x] 4.4 运行相关单测

## 5. gRPC / Loki / path params (TDD)
- [x] 5.1 先补 gRPC 错误字段名、Loki label 约束、测试路由 path param 的失败测试
- [x] 5.2 将 gRPC 错误文案收敛到 `camelCase` 字段名
- [x] 5.3 明确 Loki / Prometheus labels 保持 `snake_case` 并补约束测试
- [x] 5.4 将显式命名 path param 的测试样式改为 `camelCase`
- [x] 5.5 运行相关测试

## 6. Verification
- [x] 6.1 运行受影响 Go 测试
- [x] 6.2 运行 `openspec validate refactor-interface-naming-standards --strict --no-interactive`

## 7. Audit
- [x] 7.1 生成 `docs/plans/2026-03-06-interface-naming-audit.md`，分类说明保留的 `snake_case` 场景

## 8. Enforcement
- [x] 8.1 写入 `docs/plans/2026-03-06-interface-naming-enforcement-design.md`
- [x] 8.2 写入 `docs/plans/2026-03-06-interface-naming-enforcement.md`
- [x] 8.3 先补 `scripts/ci/check-interface-naming-test.sh` 失败测试
- [x] 8.4 新增 `scripts/ci/check-interface-naming.sh`
- [x] 8.5 接入 `.github/workflows/ci.yml`
- [x] 8.6 运行脚本回归测试与真实仓库扫描
- [x] 8.7 运行 `openspec validate refactor-interface-naming-standards --strict --no-interactive`
