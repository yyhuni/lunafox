## 1. 校验边界精简（TDD）
- [x] 1.1 Red: 新增测试，证明 Worker strict decode 会拒绝 unknown key。
- [x] 1.2 Red: 新增测试，证明 Worker strict decode 保留 required 字段存在性语义（缺字段失败）。
- [x] 1.3 Red: 新增测试，证明移除 Worker schema runtime 后核心 fail-fast 行为仍成立。
- [x] 1.4 Green: 在 typed decode 路径启用 `DisallowUnknownFields` 语义。
- [x] 1.5 Green: 在 decode 流程增加 required 字段存在性检查。
- [x] 1.6 Green: 下线或最小化 `config_schema_runtime` 在 Worker 执行链路中的职责。
- [x] 1.7 Refactor: 清理重复校验路径与错误包装，保持错误语义稳定。

## 2. config_schema_runtime 拆分防误删（TDD）
- [x] 2.1 Red: 新增测试锁定 `getConfigPath/timeoutFromSeconds` 行为不依赖 schema runtime。
- [x] 2.2 Green: 下线 schema runtime 后保留 `getConfigPath/timeoutFromSeconds` 行为。
- [x] 2.3 Refactor: 将通用函数收敛到 `config_typed.go`，减少小文件分散。

## 3. Worker 业务校验收敛（TDD）
- [x] 3.1 Red: 新增测试覆盖“保留的最小业务规则”（stage/tool 开关约束、关键必填）。
- [x] 3.2 Green: 实现最小业务规则集合并移除冗余边界校验。
- [x] 3.3 Refactor: 合并重复校验函数，提升可读性。
- [x] 3.4 Refactor: 去除 decode 后重复 `Validate()` 调用，并用回归测试锁定失败时机。

## 4. 生成链路简化（TDD）
- [x] 4.1 Red: 新增测试，验证全局生成入口可生成 schema/docs/typed 且命名稳定。
- [x] 4.2 Red: 新增测试，验证全局入口为唯一权威并移除 per-workflow 样板。
- [x] 4.3 Green: 增加全局生成命令（Makefile/脚本），减少 per-workflow 样板。
- [x] 4.4 Green: 提供 `make workflow-contracts-gen-workflow` 增量入口，移除 `contract_assets.go`。
- [x] 4.5 Refactor: 统一生成命令文档与开发约定。

## 5. 一致性与回归门禁
- [x] 5.1 运行 `go test ./worker/... -count=1`。
- [x] 5.2 运行 `go test ./server/... -count=1`（至少覆盖 preset/schema gate）。
- [x] 5.3 运行 `make test-metadata` 并确认 contract/schema/docs/typed 一致。
- [x] 5.4 运行 `openspec validate refactor-worker-workflow-lean-code-first-v2 --strict --no-interactive`。

## 6. 推广准备（不在本次实现）
- [x] 6.1 输出“新增 workflow 最小骨架”模板说明。
- [x] 6.2 形成批量迁移 old-python workflows 的执行手册（阶段化迁移清单）。
