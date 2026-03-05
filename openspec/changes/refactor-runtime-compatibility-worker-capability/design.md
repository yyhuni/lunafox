## Context
当前系统已经实现 agent/worker 进程与容器层面的解耦，但兼容性判定仍以 `agent.version` 代表 worker 能力。该近似在演进期（灰度、回滚、镜像不一致、人工配置）会出现语义偏差。

## Goals / Non-Goals
- Goals:
  - 调度兼容性判定以 worker 实际能力为准。
  - 兼容性失败在调度阶段 fail-fast，减少执行期隐式报错。
  - 为后续新增 workflow 提供可扩展能力模型。
- Non-Goals:
  - 本次不迁移新 workflow 业务逻辑。
  - 本次不改任务分配算法（优先级/并发）本身。
  - 本次不引入运行时动态插件系统。

## Decisions
1. Worker 能力模型
- 最小可用字段：`workerVersion`。
- 预留扩展字段：`supportedWorkflows`（`workflow + apiVersion + schemaVersion` 列表）。
- 判定逻辑优先使用 `supportedWorkflows`；缺失时回退到 `workerVersion` 映射；两者都缺失则 fail-closed。

2. 兼容性 Gate 输入
- 保留 `Supports(ctx, agentID, tuple)` 入口以控制改动范围。
- `Supports` 内部改为读取 agent 对应 worker 能力快照，而非 `agent.version`。

3. 数据持久化
- 在 agent runtime 记录中新增 worker 能力字段（数据库迁移）。
- heartbeat 写库时同步更新 worker 能力字段。

4. 错误契约
- 继续使用 `WORKER_VERSION_INCOMPATIBLE`，保持上层 HTTP/gRPC 错误映射稳定。
- 当能力缺失时返回同类不兼容错误，并附带明确 message（例如 missing worker capability snapshot）。

5. 更新链路解耦（P1）
- `update_required` 事件当前只携带 agent 目标版本与 agent 镜像，worker 目标依赖本地环境变量。
- 为避免“agent 升级完成但 worker 目标未对齐”，后续扩展 update contract，下发 worker 目标（至少 `workerImageRef`，可选 `workerVersion`）。
- agent updater 以 server 下发 worker 目标为主，环境变量仅作为显式 fallback（并记录告警）。

6. 存储语义对齐（P1）
- `scan_task.version` 在 SQL 层存在但在持久化模型未使用，语义漂移会误导审计。
- 方案二选一并固定：
  - 要么接入并真实写入该字段；
  - 要么通过迁移删除该字段与注释，消除“伪字段”。
- agent 运行时版本展示建议拆分为 `agentVersion` 与 `workerVersion`，避免控制面混用单一 `version`。

7. 历史协议模型收敛（P2）
- 当前 `agentproto/types.go` 与 `agent/internal/protocol/messages.go` 仍保留旧 envelope 语义，容易与 gRPC runtime 主链路混淆。
- 后续收敛为“runtime proto 单源”，历史类型标记 deprecated 并逐步下线。

8. 版本规则单源（P1）
- `apiVersion/schemaVersion` 校验规则必须跨端一致，避免“server 拒绝但 worker 接受”或反向偏差。
- 约束来源统一到共享校验组件（或同一契约包），避免各模块自带正则逐步漂移。

9. 兼容性能力来源单轨（P0）
- 兼容性判定路径收敛为“worker capability snapshot”单一来源，不保留静态 allowlist 和 `agent.version` 硬编码兜底。
- 当 capability 不可得时统一 fail-closed，避免隐式放行。

10. workflow 目录单源（P2/P3）
- server 编排层 workflow 列表与 worker contract/registry 不再双写。
- 需要引入可注入 workflow catalog（由 schema/contract 产物驱动），降低新增 workflow 的改动面。

## Architecture Sketch
```text
agent heartbeat
  -> payload(version, workerVersion, supportedWorkflows?)
  -> server runtime mapper
  -> agent runtime store (persist worker capability snapshot)
  -> task scheduler PullTask
  -> compatibility gate checks tuple against worker capability
  -> assign or fail-fast (WORKER_VERSION_INCOMPATIBLE)
```

## TDD Strategy
1. Red:
- 新增 gate 测试：仅 `agent.version` 命中但 `workerVersion` 不兼容时必须拒绝。
- 新增能力缺失测试：无 worker 能力快照时必须拒绝（fail-closed）。
- 新增 heartbeat 映射与持久化测试：worker 能力字段能从 runtime 消息流转到存储。

2. Green:
- 增加 heartbeat 字段映射与存储。
- 替换 compatibility gate 数据源与匹配逻辑。
- 补齐 wiring 与 store port。

3. Refactor:
- 移除 `agent.version -> workflows` 的静态映射冗余路径。
- 统一错误 message 与测试夹具，减少重复构造。

## Risks / Trade-offs
- 风险：proto / migration 变更面较大。
  - 缓解：先写跨层测试（mapper + gate + pull task）再落实现。
- 风险：旧 agent 未上报新字段会导致任务不可分配。
  - 缓解：明确 fail-closed，并在日志与错误信息中指出升级要求。
- 风险：能力模型复杂化。
  - 缓解：先以 `workerVersion` 最小集落地，`supportedWorkflows` 作为可选扩展。
- 风险：解耦范围扩展后改动面过大。
  - 缓解：采用 P0/P1/P2 分层交付，先稳定调度兼容性链路，再清理升级与存储语义。

## Migration Plan
1. P0: 加 Red 测试锁定目标行为。
2. P0: 升级 runtime payload 与 mapper。
3. P0: 扩展 agent runtime 存储模型与迁移（worker capability snapshot）。
4. P0: 替换 gate 为 worker 能力判定。
5. P0: 回归 `TaskRuntimeService.PullTask` 和 gRPC 错误映射。
6. P1: 更新链路对齐 worker 目标下发，避免 agent/worker 升级漂移。
7. P1: 清理 `scan_task.version` 漂移并对齐管理面 version 语义。
8. P2: 下线历史协议重复模型（保留必要兼容窗口）。
9. 运行全量测试与 OpenSpec 严格校验。

## Decoupling Inventory（防遗漏）
1. 调度兼容性：`agent.version -> worker capability snapshot`（P0）。
2. Heartbeat 协议：增加 `workerVersion/supportedWorkflows`（P0）。
3. 更新链路：`update_required` 下发 worker 目标而非仅 agent 目标（P1）。
4. 数据漂移：`scan_task.version` 去留决策并实施（P1）。
5. 管理面语义：拆分 `agentVersion` 与 `workerVersion`（P1）。
6. 历史模型：`agentproto/protocol` 与 runtime proto 单源收敛（P2）。

## Additional Decoupling Inventory（本轮补充）
> 以下为“超出当前核心改造”的补充清单，用于防遗漏；按 P2/P3 逐步推进，避免一次性扩散风险。

1. 升级链路目标一致性（P2）
- `update_required` 目前以 agent 目标为中心，worker 目标仍可能由节点本地环境变量漂移。
- 需要将 worker 目标纳入 server 下发契约并参与幂等校验。

2. 版本语义职责分离（P2）
- `version` 目前在兼容性、升级、展示、缓存等场景复用，语义过载。
- 需要明确 `agentVersion`（控制面）与 `workerVersion`（执行面）边界。

3. `scan_task.version` 历史字段去漂移（P2）
- SQL 存在字段但代码模型未接入，属于“伪契约”。
- 需做“接入写入”或“迁移删除”二选一并固化。

4. 共享目录路径契约集中化（P2）
- `/opt/lunafox` 相关路径在 server/agent/worker/安装脚本多处硬编码。
- 需要提炼为单一运行时契约源，减少跨组件耦合。

5. runtime socket 路径契约集中化（P2）
- `/run/lunafox/worker-runtime.sock` 在多处重复约定。
- 需要统一配置入口与回退策略，避免路径漂移。

6. 扫描创建单工作流限制与执行能力解耦（P3）
- 当前创建流程强制 `len(workflowNames)==1`，限制未来多 workflow 组合能力。
- 后续需评估“编排能力”与“当前单工作流业务策略”分离。

7. 配置封装形态单一化（P3）
- 调度兼容性解析同时支持根级与 workflow 嵌套两种形态，容易形成隐式歧义。
- 需要收敛为单一规范并补迁移策略。

8. 生成产物路径耦合解耦（P3）
- 合同生成脚本当前直接耦合 worker/server 目录结构。
- 后续可引入可配置输出映射，降低拆仓/模块化改造成本。

9. 历史协议类型收敛（P3）
- `agentproto/types` 与 `agent/internal/protocol/messages` 仍保留旧 envelope 语义。
- 需要明确 deprecate 时间线并收敛到 runtime proto 单源模型。

10. 编排计划硬编码解耦（P3）
- `defaultWorkflowStages` 在 domain 层硬编码工作流顺序与并行模式，新增 workflow 需改 domain 常量与 planner。
- 后续应支持由 workflow 元数据/契约驱动编排计划，降低“新增工作流 = 改多处”的耦合。

11. workflow 激活语义解耦（P3）
- `CollectEnabledWorkflowSet` 当前按配置键存在即启用 workflow，容易与 `enabled` 开关语义冲突。
- 需统一“显式启用”规则，避免配置形态变化引发隐式任务生成。

12. 版本字符串规范化解耦（P2）
- 当前部分链路使用严格字符串相等（例如升级通知去重），对 `v1.2.3` / `1.2.3` 等格式差异敏感。
- 需引入统一版本规范化策略，避免无意义的误触发升级/不兼容判定。

13. 任务配置封装解耦（P3）
- 当前 `TaskAssignment.Config` 直接透传整段 YAML，worker 再做局部提取与解码。
- 后续可考虑 server 侧按 workflow 下发最小配置切片，减少跨层解析耦合。

14. schema 发现规则解耦（P3）
- `workflowschema.ListWorkflows()` 依赖文件命名推断 workflow/api/schema 三元组。
- 需评估改为显式清单或元数据索引，降低命名规则对运行逻辑的隐式耦合。

15. 跨端 schemaVersion 规则一致性解耦（P1）
- 当前 worker 合约校验允许 semver 扩展（pre-release/build metadata），但 scheduler compatibility 仅允许 `MAJOR.MINOR.PATCH`。
- 需要统一成单一规则与单一实现，避免同一配置在 server/worker 上出现“合法性分裂”。

16. 兼容性能力来源双轨解耦（P0）
- `TaskRuntimeService` 当前仍存在“默认静态 gate”与“agentVersion resolver”双路径，且都包含硬编码 workflow tuple。
- 需要收敛为 capability snapshot 单轨，并移除默认静态兼容映射，避免隐藏行为。

17. workflow 目录双源解耦（P2/P3）
- server domain 的 `WorkflowName/ParseWorkflowName/defaultWorkflowStages` 与 worker registry/contract 为并行信息源。
- 需要引入 catalog 适配层，将“可识别 workflow 集合”与“编排元数据”从硬编码切到可扩展来源。

18. 创建期 schema gate 与编排启用集范围对齐（P1/P2）
- 当前创建链路对 schema 的校验只关注目标 workflow 的配置片段，而任务编排启用集会读取整个 root 配置键集合。
- 存在“schema 校验通过但被 root 额外键启用其他 workflow 任务”的风险，导致 workflowNames 契约与任务计划漂移。

19. 调度错误分类与重试策略解耦（P0/P1）
- `PullTask` 对 schema 解析失败、兼容性不满足等错误统一执行 `ReleaseTaskClaim -> pending`。
- 对不可恢复错误（如配置结构非法）会形成重复拉取/释放循环；在无兼容 worker 场景下会持续重试并挤占队列。

20. 扫描生命周期推进时机解耦（P1）
- 当前扫描状态会在 PullTask 早期从 pending 提前推进到 running，再做兼容性与配置判定。
- 当后续判定失败并释放任务时，scan 可能长期停留 running 但无可执行进展，造成状态语义漂移。

21. 调度头阻塞与候选过滤解耦（P1/P2）
- 当前调度顺序是“先 claim 任务，再做兼容性/配置判定，再 release 回 pending”。
- 当队头任务对当前 agent 长期不兼容或配置不可恢复错误时，易形成同一任务反复 claim/release，阻塞后续可执行任务。

22. 全局调度优先级策略解耦（P2）
- 当前 `pullTaskSQL` 将全局优先级硬编码为 `ORDER BY stage DESC, created_at ASC`。
- 该策略把“工作流阶段编号”耦合进跨扫描调度公平性，可能导致后期阶段任务长期压制早期阶段任务，且缺少策略化配置与测试护栏。

23. Schema 身份键解耦（P2）
- 生成 schema 的 `$id` 当前仅包含 `workflow + schemaVersion`，不包含 `apiVersion`。
- 在未来多 API 大版本并存且共享 schemaVersion 时，会出现标识冲突风险（缓存、引用、文档链接语义不稳定）。

24. PullTask 业务错误与传输连接生命周期解耦（P1/P2）
- 当前 `RequestTask` 分支只要 `PullTask` 返回错误就直接 `return gRPC status`，导致整个 runtime stream 断开并触发 agent 重连退避。
- 业务层可恢复拒绝（如不兼容/配置错误）被放大为传输层故障，影响 heartbeat、配置下发与调度吞吐稳定性。

25. 调度热路径 YAML 解析与版本 tuple 预计算解耦（P1/P2）
- 当前兼容性 gate 在每次 PullTask 时从 YAML 文本动态解析 `workflow/apiVersion/schemaVersion` tuple。
- 该路径与创建期 schema gate 重复，存在规则漂移与热路径开销；应考虑在创建期固化 tuple 并在调度期直接读取。

26. `scan_task.config` 投影语义收敛（P2）
- 任务模型包含 `scan_task.config`，但创建链路默认不填，运行时主要回退到 `scan.yaml_configuration`。
- “存在字段但无稳定写入策略”会导致维护误解，需明确为正式能力（按任务切片配置）或删除冗余字段。

27. `workflowIDs`/`workflowNames` 输入契约一致性（P2）
- 创建链路调度与 schema 校验只依赖 `workflowNames`，但 `workflowIDs` 会被原样持久化且缺少与 `workflowNames` 的一致性校验。
- 这会造成“展示与审计看到的 workflowIDs”与“实际执行/校验 workflowNames”语义分离，影响可追踪性与排障。

28. 任务配置传输介质解耦（P2/P3）
- 当前任务配置通过 `CONFIG=<yaml>` 环境变量注入 worker 容器。
- 大配置存在环境变量长度上限风险，且配置文本可被容器元数据/诊断链路间接暴露；应评估文件挂载或对象引用传递。
