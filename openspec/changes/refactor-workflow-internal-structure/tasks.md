## 1. 目录与包迁移（TDD）
- [ ] 1.1 为 `workflow/schema` 与 `workflow/profile` 迁移补充/更新单测基线（先 RED）
- [ ] 1.2 迁移 `server/internal/workflowschema/*` 到 `server/internal/workflow/schema/*` 并让测试 GREEN
- [ ] 1.3 迁移 `server/internal/preset/*` 到 `server/internal/workflow/profile/*` 并让测试 GREEN
- [ ] 1.4 迁移 profile 产物目录 `server/internal/preset/presets/*` 到 `server/internal/workflow/profile/profiles/*`
- [ ] 1.5 删除旧目录，确保无双写/双读路径

## 2. 依赖与 wiring 收敛
- [ ] 2.1 替换 server 所有 import 到新路径（bootstrap/catalog/scan）
- [ ] 2.2 确认 catalog workflow/profile query adapter 行为不变
- [ ] 2.3 确认 scan schema gate 行为不变（`ListWorkflows`、`ValidateYAML`）

## 3. 生成链与文档路径收敛
- [ ] 3.1 更新 worker 生成脚本默认 server schema 输出目录
- [ ] 3.2 更新 worker Makefile 中 schema 目录常量与清理规则
- [ ] 3.3 更新 contract 生成与路径映射文档（schema/profile 产物新路径）
- [ ] 3.4 同步更新 `refactor-workflow-preset-generated-profiles` 中的目录引用，消除变更间冲突

## 4. 回归验证
- [ ] 4.1 运行 server 相关测试（catalog/scan/bootstrap/workflow）
- [ ] 4.2 运行 worker 合同生成与一致性测试
- [ ] 4.3 执行 `rg -n "internal/preset|internal/workflowschema|server/internal/preset/presets|server/internal/workflowschema" server worker docs tools`，确认无残留
- [ ] 4.4 执行 `go test ./internal/modules/catalog/... ./internal/modules/scan/... ./internal/bootstrap/... -count=1`（server）
- [ ] 4.5 执行 `make workflow-contracts-ci-check`（worker）
- [ ] 4.6 执行 `openspec validate refactor-workflow-internal-structure --strict --no-interactive`
- [ ] 4.7 执行 `openspec validate refactor-workflow-preset-generated-profiles --strict --no-interactive`
