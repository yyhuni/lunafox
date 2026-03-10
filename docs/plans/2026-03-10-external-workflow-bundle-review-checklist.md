# External Workflow Bundle Review Checklist

**Goal:** 用 speckit 风格的阶段审查方式，评估 LunaFox 的 external workflow bundle 架构是否值得进入下一步实现。

**Scope:** 本清单对应两个 OpenSpec change：
- `openspec/changes/introduce-external-workflow-bundle-runtime/`
- `openspec/changes/introduce-external-workflow-bundle-mvp/`

---

## Gate 1：总方案是否成立

目标：确认 `introduce-external-workflow-bundle-runtime` 是否被正确定位为“目标架构 / umbrella change”，而不是一次性实现清单。

### 审查项
- [ ] 总方案是否明确说明它定义的是最终目标边界，而不是单批落地任务
- [ ] 总方案是否明确说明后续必须按 phase 逐步落地，而不是直接切掉 builtin-first
- [ ] 总方案是否明确把目标收敛为“workflow bundle 全外置 + worker-core plugin host”
- [ ] 总方案是否保留 worker-core 对工具执行、日志、超时、取消、资源控制的宿主权威
- [ ] 总方案是否明确禁止 in-process Go plugin 动态加载

### 重点文件
- `openspec/changes/introduce-external-workflow-bundle-runtime/proposal.md:20`
- `openspec/changes/introduce-external-workflow-bundle-runtime/proposal.md:32`
- `openspec/changes/introduce-external-workflow-bundle-runtime/design.md:14`
- `openspec/changes/introduce-external-workflow-bundle-runtime/design.md:50`
- `openspec/changes/introduce-external-workflow-bundle-runtime/design.md:107`

### 审查结论模板
- 通过：总方案被正确定位为目标架构，未把长期愿景误写成首批实现任务。
- 不通过：总方案范围过大，仍然混入过多首批实现承诺，需要继续收敛。

---

## Gate 2：MVP 是否足够收敛

目标：确认 `introduce-external-workflow-bundle-mvp` 只聚焦“治理 + 握手 + 兼容门禁”，没有偷偷膨胀成完整插件系统。

### 审查项
- [ ] MVP 是否只覆盖 external bundle metadata 治理
- [ ] MVP 是否只覆盖 plugin bootstrap handshake（而不是完整执行协议）
- [ ] MVP 是否只覆盖 capability snapshot 与 scheduler fail-closed 兼容门禁
- [ ] MVP 是否明确不做 `Execute / Cancel` 真实执行链路
- [ ] MVP 是否明确不做完整 marketplace / registry 运营能力
- [ ] MVP 是否明确允许 builtin 与 external 双轨并存

### 重点文件
- `openspec/changes/introduce-external-workflow-bundle-mvp/proposal.md:18`
- `openspec/changes/introduce-external-workflow-bundle-mvp/proposal.md:33`
- `openspec/changes/introduce-external-workflow-bundle-mvp/design.md:9`
- `openspec/changes/introduce-external-workflow-bundle-mvp/design.md:21`
- `openspec/changes/introduce-external-workflow-bundle-mvp/design.md:78`

### 审查结论模板
- 通过：MVP 范围足够小，可作为第一阶段架构验证。
- 不通过：MVP 仍然混入过多 Phase 2/3 内容，需继续拆分。

---

## Gate 3：事实源与边界是否清楚

目标：确认 builtin workflow 与 external workflow bundle 的事实源没有混淆，避免后续出现双重真相。

### 审查项
- [ ] 是否明确 builtin workflow 的事实源仍然是 repo 内 contract definition
- [ ] 是否明确 external workflow 的事实源是 imported bundle metadata + verified artifact record
- [ ] 是否明确 server / worker-core / plugin 三方职责边界
- [ ] 是否明确 catalog 真相、artifact 真相、install reference 真相的归属
- [ ] 是否明确 external bundle 不直接绕过 worker-core 控制工具执行

### 重点文件
- `openspec/changes/introduce-external-workflow-bundle-mvp/design.md:21`
- `openspec/changes/introduce-external-workflow-bundle-runtime/design.md:61`
- `openspec/changes/introduce-external-workflow-bundle-runtime/design.md:71`
- `openspec/changes/introduce-external-workflow-bundle-runtime/design.md:94`

### 审查结论模板
- 通过：两类 workflow 的事实源分离清楚，边界清晰。
- 不通过：builtin / external 的 truth source 和职责边界仍有重叠或歧义。

---

## Gate 4：兼容性模型是否够硬

目标：确认 external bundle 引入后，系统仍然以 capability snapshot 为主，并继续 fail-closed。

### 审查项
- [ ] external bundle tuple 是否进入 worker capability snapshot
- [ ] tuple 是否至少包含 `workflowId / executorType / bundleDigest / pluginApiVersion`
- [ ] scheduler 是否继续以 capability snapshot 为准，而不是 agent/version 猜测
- [ ] snapshot 缺失时是否继续 fail-closed
- [ ] builtin 与 external 双轨并存时是否有明确兼容判定优先级

### 重点文件
- `openspec/changes/introduce-external-workflow-bundle-mvp/design.md:45`
- `openspec/changes/introduce-external-workflow-bundle-mvp/specs/runtime-worker-compatibility/spec.md:3`
- `openspec/changes/introduce-external-workflow-bundle-mvp/specs/runtime-worker-compatibility/spec.md:17`

### 审查结论模板
- 通过：兼容门禁继续保持 capability-first、fail-closed，不会因外置 bundle 放松约束。
- 不通过：external tuple 设计不足，或仍然依赖版本号猜测兼容性。

---

## Gate 5：插件握手模型是否可演进

目标：确认 Phase 1 的 plugin bootstrap 足够小，但不会堵死后续 `Validate / Execute / Cancel` 扩展。

### 审查项
- [ ] 是否固定了最小握手操作：`GetPluginInfo / Capabilities / Health`
- [ ] 是否明确 plugin 必须在隔离边界外运行
- [ ] 是否明确 `Health` 不是单一布尔值，而是条件集合
- [ ] 是否给后续 `Validate / Execute / Cancel` 预留了扩展空间
- [ ] 是否没有把 transport 细节写死到后续不可演进

### 重点文件
- `openspec/changes/introduce-external-workflow-bundle-mvp/specs/workflow-plugin-bootstrap/spec.md:3`
- `openspec/changes/introduce-external-workflow-bundle-mvp/specs/workflow-plugin-bootstrap/spec.md:12`
- `openspec/changes/introduce-external-workflow-bundle-mvp/specs/workflow-plugin-bootstrap/spec.md:21`

### 审查结论模板
- 通过：握手模型足够小，且可平滑演进到完整 runtime protocol。
- 不通过：当前 bootstrap 协议要么过重，要么缺少后续扩展位。

---

## Gate 6：控制面治理是否足够严谨

目标：确认 external bundle 在进入调度之前，已经具备最低限度的治理状态机与身份信息。

### 审查项
- [ ] server 是否只暴露 `activated` 的 bundle version
- [ ] bundle metadata 是否至少包含 digest / pluginApiVersion / workerVersionRange
- [ ] import / verify / activate / revoke 状态是否足以支撑后续扩展
- [ ] 是否没有出现“导入即执行”的危险路径
- [ ] 是否为后续 trust / signature / allowlist 留下接口位置

### 重点文件
- `openspec/changes/introduce-external-workflow-bundle-mvp/specs/external-workflow-bundle-catalog/spec.md:3`
- `openspec/changes/introduce-external-workflow-bundle-mvp/specs/external-workflow-bundle-catalog/spec.md:18`
- `openspec/changes/introduce-external-workflow-bundle-runtime/design.md:107`

### 审查结论模板
- 通过：control plane 已形成最小治理闭环。
- 不通过：bundle metadata 或生命周期状态不足以支撑调度前治理。

---

## 建议优先确认的 4 个问题

- [ ] `bundleDigest` 是否要作为调度兼容 tuple 的强约束，而不是仅作展示元数据
- [ ] `Health` 是否固定为 `Ready / Degraded / NotReady` 三态条件模型
- [ ] Phase 1 的 bootstrap transport 是否先只声明“隔离协议边界”，暂不锁死 gRPC 或 STDIO
- [ ] `revoked` bundle 对正在运行任务的语义是否留到下一阶段单独定义

---

## 最终审查结论

### 总方案
- [ ] 通过
- [ ] 不通过
- 备注：

### MVP
- [ ] 通过
- [ ] 不通过
- 备注：

### 下一步建议
- [ ] 可以进入 Phase 1 实施计划
- [ ] 需要先修改 OpenSpec 文档后再审
- [ ] 需要继续补充兼容性 / 安全 / 生命周期细节
