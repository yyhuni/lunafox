# Backend 综合代码审查报告（V2 执行版）

**修订日期**: 2026-02-14  
**审查范围**: `server/`、`worker/`、`agent/`  
**目标**: 给出“能直接排期执行”的结论，减少二次讨论成本。  
**结论口径**: 以当前代码状态为准（非历史快照）。

---

## 1. 执行摘要（先看这里）

当前最关键结论：

1. 原“确认问题”中的 6 条，已基本完成修复并落地。  
2. 新增 1 条应升级为**确认缺陷**（`agent Executor` 的 `WaitGroup` 使用时序）。  
3. 4 条“待定问题”中：2 条建议升级处理，2 条维持观测项。  
4. 其余优化建议不建议与缺陷修复混排，应纳入技术债迭代。

**建议排期**：

1. 本周（P0/P1）：修 `Executor` 并发缺陷 + Docker 清理超时保护。  
2. 下周（P2）：补 `BatchSender` 队列上限与指标。  
3. 持续：保留性能优化项，基于压测数据再动刀。

---

## 2. 总览矩阵（可直接用于排期）

| ID | 项目 | 当前状态 | 风险等级 | 是否必须修 | 建议优先级 |
|---|---|---|---|---|---|
| C1 | Subdomain 解析输出 channel 阻塞 | 已修复 | 中 | 否（已完成） | 已完成 |
| C2 | Wordlist 行数错误吞掉并写 0 | 已修复 | 高 | 否（已完成） | 已完成 |
| C3 | Redis ping 失败未显式 close | 已修复 | 中 | 否（已完成） | 已完成 |
| C4 | `buildDependencies()` 复杂度过高 | 已修复（结构拆分） | 中 | 否（已完成） | 已完成 |
| C5 | `runner.go` 复杂度过高 | 已修复（职责拆分） | 中 | 否（已完成） | 已完成 |
| C6 | `doc-gen/main.go` 复杂度过高 | 已修复（分层+测试） | 低 | 否（已完成） | 已完成 |
| N1 | `Executor` WaitGroup Add/Wait 时序风险 | **确认缺陷** | **高** | **是** | **P0** |
| D1 | 删除扫描时吞部分 domain 错误语义 | 待业务定案 | 中 | 建议小修 | P2 |
| D2 | Wordlist 扫描器 64KB 缓冲疑虑 | 已转“约束策略” | 低 | 否 | 关闭 |
| D3 | BatchSender 失败回队策略 | 观测项（可增强） | 中 | 建议增强 | P2 |

---

## 3. 已完成修复项（6 条）

### C1) Subdomain 解析输出通道可能阻塞
- **文件**: `/Users/yangyang/Desktop/lunafox/worker/internal/results/subdomain_parser.go`
- **现状**: `ParseSubdomains` 已引入 `context.Context`，发送时使用 `select` 监听 `ctx.Done()`，上游取消可及时收敛。
- **优点**:
1. 生产协程不再无限阻塞在 `out <- item`。
2. 任务取消时能尽快退出，减少 goroutine 堆积。
- **残余风险**:
1. `seen` map 在超大结果集下仍有峰值内存压力（这是容量问题，不是逻辑 bug）。
- **是否继续修**: 否（当前可接受）。
- **后续建议**: 若压测触发 OOM，再评估分片去重/外部去重策略。

### C2) Wordlist 行数错误被吞掉并写 0
- **文件**: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/application/local_wordlist_file_store.go`
- **现状**: 行数统计失败会返回错误，保存流程不再“假成功 + lineCount=0”。
- **优点**:
1. 元数据可靠性恢复。
2. 避免下游基于错误行数做错误决策。
- **残余风险**:
1. 严格模式下，异常输入会被拒绝（这是预期行为）。
- **是否继续修**: 否（已达标）。

### C3) Redis ping 失败时未显式关闭
- **文件**: `/Users/yangyang/Desktop/lunafox/server/internal/bootstrap/infra.go`
- **现状**: `Ping` 失败后先 `Close()`，并记录 close 异常日志。
- **优点**:
1. 初始化失败路径资源释放更完整。
2. 运维排查有日志可追踪。
- **残余风险**: 无显著风险。
- **是否继续修**: 否。

### C4) `buildDependencies()` 复杂度过高
- **文件**: `/Users/yangyang/Desktop/lunafox/server/internal/bootstrap/wiring.go`
- **现状**: 主流程以模块编排为主，细节分发到模块 wiring。
- **优点**:
1. 接入新模块时影响面更可控。
2. 故障定位能按模块切片排查。
- **残余风险**:
1. 单文件 import 仍偏重，但已显著优于单巨函数。
- **是否继续修**: 否（非阻断）。

### C5) `runner.go` 复杂度过高
- **文件**:
  - `/Users/yangyang/Desktop/lunafox/worker/internal/activity/runner.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/activity/runner_execution.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/activity/runner_output.go`
- **现状**: 已按“执行生命周期/输出处理/公共结构”拆分。
- **优点**:
1. 错误路径更容易做单测覆盖。
2. 输出处理逻辑可独立演进。
- **残余风险**:
1. 极端清理路径（外部依赖阻塞）仍可继续增强超时保护。
- **是否继续修**: 否（主问题已解决）。

### C6) `doc-gen/main.go` 复杂度过高
- **文件**:
  - `/Users/yangyang/Desktop/lunafox/worker/cmd/doc-gen/main.go`
  - `/Users/yangyang/Desktop/lunafox/worker/cmd/doc-gen/loader.go`
  - `/Users/yangyang/Desktop/lunafox/worker/cmd/doc-gen/renderer.go`
  - `/Users/yangyang/Desktop/lunafox/worker/cmd/doc-gen/writer.go`
  - `/Users/yangyang/Desktop/lunafox/worker/cmd/doc-gen/types.go`
  - `/Users/yangyang/Desktop/lunafox/worker/cmd/doc-gen/doc_gen_test.go`
- **现状**: 已完成 loader/renderer/writer 抽离，`main` 只编排，且补了覆盖关键分支的测试。
- **优点**:
1. 变更耦合大幅下降。
2. 回归成本可控。
- **残余风险**: 无明显风险。
- **是否继续修**: 否。

---

## 4. 新增确认缺陷（必须修）

### N1) `Executor` 的 `WaitGroup` Add/Wait 存在并发时序风险
- **文件**: `/Users/yangyang/Desktop/lunafox/agent/internal/task/executor.go`
- **问题描述**:
1. `Start()` 中先 `go e.execute(...)`。
2. `execute()` 内部才 `e.wg.Add(1)`。
3. `Shutdown()` 同时调用 `e.wg.Wait()`。

这会导致 `Add` 和 `Wait` 并发交错，存在 `WaitGroup` 误用风险（可能出现等待遗漏或 panic，取决于时序）。

- **优点（当前实现）**:
1. 有 `Shutdown(ctx)` + `CancelAll()` + `Wait()` 的完整框架。
2. 任务取消流程设计方向正确。

- **缺点/风险**:
1. 并发时序不安全，属于基础并发正确性问题。
2. 在高并发或停机窗口更容易暴露。

- **修复方式（建议一次到位）**:
1. 在 `Start()` 启 goroutine 前先 `e.wg.Add(1)`。
2. 用 goroutine 包装 `execute()` 并 `defer e.wg.Done()`。
3. `execute()` 内删除 `wg.Add/Done`。
4. 增加并发关闭场景单测，并跑 `go test -race`。

- **是否修复**: **是（必须）**。  
- **优先级**: **P0**。

---

## 5. 待定问题复核结论（给出是否要修）

### D1) 扫描删除流程“吞错误”是否合理
- **文件**: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_lifecycle_service.go`
- **现状**: 删除场景下，`ErrScanCannotStop` / `ErrInvalidStatusChange` 被转为 `nil`，删除继续。
- **优点**:
1. 删除接口幂等性更好，不因状态漂移阻塞删除。
- **风险**:
1. 可观测性下降，难还原“为什么没真正 stop”。
- **建议修复**:
1. 保留当前业务语义不改。
2. 增加 warning 日志 + 指标（如 `scan_delete_stop_ignored_total`）。
- **是否修复**: 建议修（小改，非阻断）。
- **优先级**: P2。

### D2) Wordlist 扫描器默认缓冲区是否是问题
- **文件**: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/application/local_wordlist_file_store.go`
- **现状**: 已不依赖 `Scanner` 行分词，改为字节流计数，并且明确单行最大 64KB 的强约束。
- **优点**:
1. 行长控制明确、可预测。
2. 避开 `Scanner` token 限制语义歧义。
- **风险**:
1. 超长行会被拒绝（属于策略）。
- **是否修复**: 否，建议关闭该待定项。

### D3) BatchSender 失败回队策略是否要重构
- **文件**: `/Users/yangyang/Desktop/lunafox/worker/internal/server/batch_sender.go`
- **现状**: 发送失败后 `append(toSend, s.batch...)` 回队，错误上抛调用方。
- **优点**:
1. 不丢数据、顺序保持直观。
2. 与上层错误处理链路兼容。
- **风险**:
1. 若上层持续重试且持续失败，内存队列可能增长。
- **建议修复**:
1. 增加队列上限 `maxQueuedItems`（超限快速失败）。
2. 增加观测指标：当前队列长度、重试次数、最终丢弃次数。
- **是否修复**: 建议修（增强项）。
- **优先级**: P2。

---

## 6. 技术债优化项（不与缺陷混排）

以下继续保留为技术债，不建议本轮“立刻改”：

1. 各类 mapper 的拷贝/分配优化。  
2. 查询层 `Count + Find` 热点分页优化。  
3. WebSocket/Hub 的统一关闭路径进一步收敛。  
4. 构造函数参数对象化（降低签名膨胀）。  

执行原则：

1. 没有压测证据，不做“先验优化”。  
2. 有明确 CPU/内存/延迟收益，再进排期。  

---

## 7. 推荐排期（可直接抄到迭代计划）

### Sprint A（本周）

1. 修复 N1：`Executor` WaitGroup 并发时序（P0）。  
2. 给 `Executor` 的 Docker `Stop/Remove` 路径增加超时 context（P1）。  

**验收标准**:

1. 并发停机压测下无 panic。  
2. `go test -race ./agent/internal/task/...` 通过。  
3. Shutdown 在外部依赖阻塞时可按超时返回，不无限挂起。  

### Sprint B（下周）

1. D1 可观测性增强：删除流程忽略 stop 错误时打日志+指标（P2）。  
2. D3 批量发送增强：回队上限+指标（P2）。  

**验收标准**:

1. 失败场景下可观测字段完整（scanID/taskID/error type）。  
2. 压测中无异常队列膨胀。  

---

## 8. 建议补充的回归检查命令

```bash
# Worker
cd /Users/yangyang/Desktop/lunafox/worker
go test ./internal/activity ./internal/results ./internal/workflow/subdomain_discovery ./cmd/doc-gen

# Server
cd /Users/yangyang/Desktop/lunafox/server
go test ./internal/modules/catalog/application ./internal/modules/scan/application ./internal/bootstrap

# Agent（重点）
cd /Users/yangyang/Desktop/lunafox/agent
go test -race ./internal/task/...
```

---

## 9. 结论（供决策）

当前代码质量相较初版审计已明显改善，主要堵点已从“普遍结构问题”收敛到“少量并发正确性与可观测性细节”。  
下一步最关键动作只有一个：**先修 `Executor` 的 WaitGroup 并发时序问题**。修完后，系统级稳定性风险会再下降一个量级。

