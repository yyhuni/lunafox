## 1. Contract executor binding model (TDD)
- [ ] 1.1 为 contract 定义增加失败测试，验证 workflow 必须显式声明 executor binding（RED）
- [ ] 1.2 运行 `go test ./internal/workflow/... -run Executor -count=1`（worker），确认测试先失败
- [ ] 1.3 最小实现 contract executor binding 类型与 subdomain_discovery 的 builtin binding（GREEN）
- [ ] 1.4 运行 worker contract 相关测试，确认通过

## 2. Manifest generation (TDD)
- [ ] 2.1 为 manifest 生成器增加测试，验证 manifest 含 `executor.type/ref`（RED）
- [ ] 2.2 运行 `go test ./cmd/workflow-contract-gen -run Manifest -count=1`（worker），确认测试先失败
- [ ] 2.3 最小实现 executor binding 生成到 manifest（GREEN）
- [ ] 2.4 重新生成 artifacts 并确认 worker/server manifest 一致

## 3. Worker builtin executor resolution (TDD)
- [ ] 3.1 为 worker 执行入口增加测试，验证 builtin binding 使用 `ref` 解析 workflow（RED）
- [ ] 3.2 增加 unsupported executor type 被显式拒绝的测试（RED）
- [ ] 3.3 最小实现 worker 侧 executor binding 解析与 builtin registry 映射（GREEN）
- [ ] 3.4 运行 `go test ./cmd/worker ./internal/workflow/... -count=1`（worker），确认通过

## 4. Server bundle metadata compatibility
- [ ] 4.1 更新 manifest types/loader，使 server 能解析 executor binding
- [ ] 4.2 运行 server workflow manifest / catalog 相关测试，确认不回归

## 5. Validation
- [ ] 5.1 运行 `./scripts/gen-workflow-contracts.sh`（worker）
- [ ] 5.2 运行 `go test ./internal/workflow/...`（worker）
- [ ] 5.3 运行 `go test ./internal/workflow/... ./internal/modules/catalog/...`（server）
- [ ] 5.4 运行 `openspec validate introduce-workflow-executor-binding --strict --no-interactive`
