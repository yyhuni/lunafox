## 1. Manifest slimming (TDD)
- [ ] 1.1 为 manifest 生成器增加失败测试，验证参数约束类字段不会继续输出到 manifest（RED）
- [ ] 1.2 运行 `go test ./cmd/workflow-contract-gen -run Manifest -count=1`（worker），确认测试先失败
- [ ] 1.3 最小实现 manifest 生成字段瘦身（GREEN）
- [ ] 1.4 运行相关生成器测试，确认通过

## 2. Generated default profile (TDD)
- [ ] 2.1 为 default profile 生成增加测试，验证 contract 默认值可生成完整 profile（RED）
- [ ] 2.2 运行 `go test ./internal/workflow/... -run Profile -count=1`（worker/server），确认测试先失败
- [ ] 2.3 最小实现 default profile 生成逻辑（GREEN）
- [ ] 2.4 运行 schema 校验测试，确认 generated default profile 合法

## 3. Sparse scenario overlays (TDD)
- [ ] 3.1 增加 overlay 合并测试，验证 scenario profile 只需表达差异字段（RED）
- [ ] 3.2 最小实现 overlay merge 生成最终 profile artifact（GREEN）
- [ ] 3.3 运行相关 profile/loader 测试，确认通过

## 4. Server consumption compatibility
- [ ] 4.1 运行 catalog / scan create 相关测试，确保 manifest/schema/profile 瘦身后 server 仍能正常消费
- [ ] 4.2 如有依赖 manifest 参数元信息的消费点，调整为依赖 contract/schema 的稳定投影

## 5. Generated artifact policy
- [ ] 5.1 增加 CI / 脚本校验，确保 generated artifacts 与 source definition 无漂移
- [ ] 5.2 运行 `openspec validate simplify-workflow-artifact-responsibilities --strict --no-interactive`
