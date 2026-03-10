## 1. Design Alignment
- [ ] 1.1 明确 manifest 顶层结构与最小必填字段（包含 `defaultProfileId`）
- [ ] 1.2 明确 manifest / schema / profile 三类工件的单向映射规则
- [ ] 1.3 明确与 `refactor-workflow-id-semantics`、`refactor-workflow-contract-naming`、`add-workflow-config-defaulting` 的 rebase 规则

## 2. Generator & Contracts
- [ ] 2.1 在 workflow contract 层补充 manifest 所需元数据与约束表达
- [ ] 2.2 扩展 generator 输出 `*.manifest.json`
- [ ] 2.3 扩展 generator 输出 manifest 中的 `defaultProfileId` 与独立 profile artifact 引用
- [ ] 2.4 重构 generator 输出的 schema，移除所有 workflow 业务扩展字段

## 3. Server Loading Split
- [ ] 3.1 拆分 server 侧 schema validation 与 manifest metadata loading 职责
- [ ] 3.2 为 manifest loader 实现 strict decode、unknown field reject 与显式语义校验
- [ ] 3.3 将 catalog / metadata 查询切换为读取 manifest
- [ ] 3.4 为 manifest loader 补充缺失字段、重复 workflowId、非法 stage/tool 结构与错误 profile 引用测试

## 4. Model Separation
- [ ] 4.1 拆分 workflow manifest 模型与 activity template metadata 模型
- [ ] 4.2 清理 `worker/internal/workflow/workflow_metadata.go` 与 activity template 对 manifest 语义的误用
- [ ] 4.3 为两套模型分别补充边界测试

## 5. Defaulting Rebase
- [ ] 5.1 将 workflow 默认值补齐逻辑改为基于 manifest / contract
- [ ] 5.2 将 canonical workflow YAML 持久化逻辑改为基于 manifest / contract 归一化结果
- [ ] 5.3 保证 worker 与 server 对短配置的归一化行为一致

## 6. Documentation & Validation
- [ ] 6.1 更新 workflow 生成文档与 schema README
- [ ] 6.2 同步改写相关 OpenSpec / docs plans 基线
- [ ] 6.3 重生成 schema / manifest / profile / docs 产物
- [ ] 6.4 运行相关测试与 `openspec validate --strict --no-interactive`
