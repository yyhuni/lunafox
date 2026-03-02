## 1. Runtime 协议与能力模型（TDD）
- [x] 1.1 Red: 新增测试，验证 heartbeat 仅有 `version` 时不会被视为 worker 能力已就绪。
- [x] 1.2 Red: 新增测试，验证 heartbeat 包含 `workerVersion` 时可正确映射到应用层输入。
- [x] 1.3 Green: 扩展 runtime 协议字段并更新生成代码。
- [x] 1.4 Green: 更新 mapper 与 heartbeat 处理链路。
- [x] 1.5 Refactor: 清理重复转换代码，统一 payload 校验入口。

## 2. Server 存储与端口改造（TDD）
- [x] 2.1 Red: 新增 repository/application 测试，验证 worker 能力字段可持久化与读取。
- [x] 2.2 Green: 增加数据库迁移与模型字段。
- [x] 2.3 Green: 扩展 `TaskRuntimeAgentRecord` / store adapter 提供 worker 能力。
- [x] 2.4 Refactor: 收敛兼容性字段命名，避免 `agent version` 与 `worker version` 语义混淆。

## 3. Compatibility Gate 解耦（TDD）
- [x] 3.1 Red: 新增测试，验证 `agent.version` 匹配但 `workerVersion` 不匹配时必须拒绝。
- [x] 3.2 Red: 新增测试，验证缺失 worker 能力快照时 fail-closed。
- [x] 3.3 Red: 新增测试，验证 worker 能力匹配时可分配任务。
- [x] 3.4 Green: 替换 `agentVersionCompatibilityGate` 为 worker 能力 gate。
- [x] 3.5 Refactor: 删除旧静态映射分支与死代码。

## 4. 调度链路与错误契约回归（TDD）
- [x] 4.1 Red: 新增 `PullTask` 端到端测试，验证不兼容时释放 claim 且返回 `WORKER_VERSION_INCOMPATIBLE`。
- [x] 4.2 Green: 调整 `TaskRuntimeService` 错误 message，明确 worker capability 缺失/不兼容原因。
- [x] 4.3 Refactor: 统一 gRPC 错误映射断言，减少重复 case。

## 5. 验证与门禁
- [x] 5.1 运行 `go test ./server/internal/modules/agent/... -count=1`。
- [x] 5.2 运行 `go test ./server/internal/modules/scan/... -count=1`。
- [x] 5.3 运行 `go test ./server/internal/grpc/runtime/service/... -count=1`。
- [x] 5.4 运行 `go test ./agent/internal/runtime/... -count=1`。
- [x] 5.5 运行 `openspec validate refactor-runtime-compatibility-worker-capability --strict --no-interactive`。

## 6. 更新链路解耦（P1, TDD）
- [x] 6.1 Red: 新增测试，验证 `update_required` 仅下发 agent 目标时会提示 worker 目标缺失风险。
- [x] 6.2 Red: 新增测试，验证下发 `workerImageRef` 后 agent updater 优先使用该目标。
- [x] 6.3 Green: 扩展 update contract 并更新 server publisher / agent client / updater。
- [x] 6.4 Refactor: 收敛 `AGENT_VERSION` 与 `WORKER_IMAGE_REF` 更新路径，统一日志语义。

## 7. 存储与管理面语义对齐（P1, TDD）
- [x] 7.1 Red: 新增测试，锁定 `scan_task.version` 的真实行为（写入或不存在）。
- [x] 7.2 Green: 执行二选一方案（启用并写入该字段，或迁移删除该字段）。
- [x] 7.3 Red: 新增管理面测试，区分 `agentVersion` 与 `workerVersion` 展示语义。
- [x] 7.4 Green: 调整 DTO/查询映射与缓存模型，避免单一 `version` 混用。

## 8. 历史协议模型收敛（P2）
- [x] 8.1 Red: 新增测试，确保 runtime 主链路不依赖 `agentproto/protocol` 历史 envelope。
- [x] 8.2 Green: 标记历史类型 deprecated，迁移调用方到 runtime proto 单源。
- [x] 8.3 Refactor: 清理重复类型与注释，补齐迁移文档。

## 9. 运行时路径契约解耦（P2, TDD）
- [x] 9.1 Red: 新增测试，锁定 `/opt/lunafox` 共享路径在 server/agent/worker 的一致性要求。
- [x] 9.2 Red: 新增测试，锁定 runtime socket 路径配置与默认值行为一致性。
- [x] 9.3 Green: 提炼共享路径与 socket 路径为单一契约来源（配置/常量/注入）。
- [x] 9.4 Refactor: 清理重复硬编码路径，补统一注释与文档。

## 10. 扫描配置封装规范化（P3, TDD）
- [x] 10.1 Red: 新增测试，固定“配置仅允许单一封装形态”预期行为。
- [x] 10.2 Green: 收敛解析逻辑，去除根级与 workflow 嵌套双路径歧义。
- [x] 10.3 Refactor: 输出迁移提示与兼容窗口策略。

## 11. 扫描编排能力与单引擎限制解耦（P3）
- [x] 11.1 Red: 新增测试，分离“业务策略单引擎”与“框架能力可多引擎”的语义边界。
- [x] 11.2 Green: 设计并实现可扩展编排接口（不要求本轮开放多引擎执行）。
- [x] 11.3 Refactor: 清理 `len(engineNames)==1` 的隐式耦合注释与错误提示。

## 12. 生成产物路径耦合解耦（P3）
- [x] 12.1 Red: 新增测试，验证 contract 生成支持可配置输出映射，不依赖固定目录布局。
- [x] 12.2 Green: 调整生成脚本/Makefile 参数模型，支持解耦输出路径。
- [x] 12.3 Refactor: 更新开发文档与 CI 门禁命令，保证迁移后仍可一键生成。

## 13. 编排计划与激活语义解耦（P3, TDD）
- [x] 13.1 Red: 新增测试，锁定新增 workflow 不应依赖改动 domain 硬编码 stage 列表。
- [x] 13.2 Red: 新增测试，锁定 workflow 激活必须遵循显式 `enabled` 语义。
- [x] 13.3 Green: 将编排计划来源从硬编码迁移到可扩展元数据/契约映射。
- [x] 13.4 Refactor: 清理 `CollectEnabledWorkflowSet` 的隐式启用路径与歧义注释。

## 14. 版本规范化与任务配置封装解耦（P2/P3, TDD）
- [x] 14.1 Red: 新增测试，验证 `vX.Y.Z` 与 `X.Y.Z` 在升级判定中的等价行为。
- [x] 14.2 Green: 引入统一版本规范化策略并应用于升级/兼容判定链路。
- [x] 14.3 Red: 新增测试，验证任务下发支持 workflow 最小配置切片。
- [x] 14.4 Green: 调整 `TaskAssignment` 配置封装，减少 worker 二次解析整段 YAML。

## 15. engineschema 发现规则解耦（P3）
- [x] 15.1 Red: 新增测试，锁定 engine 发现不依赖文件命名推断副作用。
- [x] 15.2 Green: 引入显式 schema 索引/元数据驱动发现机制。
- [x] 15.3 Refactor: 保留兼容窗口并输出迁移告警。

## 16. 版本规则单源与跨端一致性（P1, TDD）
- [x] 16.1 Red: 新增测试，验证 scheduler compatibility 与 worker contract 对 `schemaVersion` 的判定规则完全一致（含 pre-release/build metadata）。
- [x] 16.2 Red: 新增测试，验证 `apiVersion/schemaVersion` 非法值在 server/worker 两侧给出一致的拒绝语义。
- [x] 16.3 Green: 提炼共享版本校验组件并替换各处本地正则实现。
- [x] 16.4 Refactor: 统一版本格式错误 message，减少跨层错误语义漂移。

## 17. Compatibility Gate 能力来源单轨化（P0/P1, TDD）
- [x] 17.1 Red: 新增测试，验证未注入 capability 数据源时不得回退静态 allowlist，必须 fail-closed。
- [x] 17.2 Red: 新增测试，锁定生产路径不允许硬编码 workflow tuple 映射参与判定。
- [x] 17.3 Green: 移除默认静态 gate 与 `agent.version` 硬编码 resolver，统一 capability snapshot 判定入口。
- [x] 17.4 Refactor: 收敛 `TaskRuntimeService` 构造函数语义，避免多构造器隐式策略差异。

## 18. workflow 目录与编排元数据单源化（P2/P3, TDD）
- [x] 18.1 Red: 新增测试，验证 server 可识别 workflow 集与 worker registry/schema catalog 一致。
- [x] 18.2 Red: 新增测试，验证新增 workflow 时无需修改 `ParseWorkflowName` 硬编码 switch 即可被 catalog 识别。
- [x] 18.3 Green: 引入 workflow catalog 适配层，替代 domain 层重复常量/解析逻辑的直接耦合。
- [x] 18.4 Refactor: 清理双写常量与兼容桥接代码，补迁移注释和演进边界。

## 19. 创建期 schema gate 与编排启用集范围对齐（P1/P2, TDD）
- [x] 19.1 Red: 新增测试，验证 root 存在额外 workflow 键时，不得在 `engineNames` 之外生成任务计划。
- [x] 19.2 Red: 新增测试，验证 `engine` 嵌套键存在但非 object 时，创建阶段直接失败而非回退到 root 解析。
- [x] 19.3 Green: 统一创建链路配置提取规则，确保 schema 校验输入与任务编排输入同源。
- [x] 19.4 Refactor: 收敛 `CollectEnabledWorkflowSet` 与创建链路启用策略，避免键存在即启用的隐式语义。

## 20. 调度错误分类与重试抖动治理（P0/P1, TDD）
- [x] 20.1 Red: 新增测试，验证不可恢复错误（schema/config 结构错误）不会被回退为 pending 无限重试。
- [x] 20.2 Red: 新增测试，验证“当前无兼容 worker”场景不会导致同一任务被同一 agent 高频反复 claim/release。
- [x] 20.3 Red: 新增测试，验证兼容性/配置判定失败后 scan 状态不会被错误推进为 running。
- [x] 20.4 Green: 引入可恢复/不可恢复错误分类策略，调整 claim 释放与终态写入路径。
- [x] 20.5 Refactor: 统一调度失败日志与指标标签，补充重试/抖动可观测性。

## 21. 调度头阻塞与候选过滤（P1/P2, TDD）
- [x] 21.1 Red: 新增测试，验证队头任务不兼容时不会无限阻塞后续可执行任务分配。
- [x] 21.2 Red: 新增测试，验证同一 agent 在短窗口内不会重复 claim 同一已判定不兼容任务。
- [x] 21.3 Green: 引入候选过滤/回避机制（如黑名单窗口、重试预算或可恢复性标记）打破 claim/release 循环。
- [x] 21.4 Refactor: 增加调度可观测字段（拒绝原因、重试次数、最后拒绝时间）并统一日志。

## 22. 全局调度优先级策略显式化（P2, TDD）
- [x] 22.1 Red: 新增测试，锁定跨扫描调度公平性预期（避免后期 stage 任务长期压制早期任务）。
- [x] 22.2 Red: 新增测试，锁定调度顺序策略变更时不影响同一扫描 stage 依赖约束。
- [x] 22.3 Green: 将 `pullTaskSQL` 的优先级规则抽象为显式策略（可配置或可替换），去除硬编码 `stage DESC` 耦合。
- [x] 22.4 Refactor: 补齐调度策略文档与回归测试，明确默认策略与演进边界。

## 23. Schema 身份键稳定性（P2, TDD）
- [x] 23.1 Red: 新增测试，验证 schema `$id` 在 `workflow + apiVersion + schemaVersion` 维度唯一。
- [x] 23.2 Red: 新增测试，验证 server/worker 生成产物对 `$id` 规则一致。
- [x] 23.3 Green: 调整 schema 生成器与一致性测试，纳入 `apiVersion` 到 `$id`。
- [x] 23.4 Refactor: 更新文档与引用规范，避免旧 `$id` 语义继续扩散。

## 24. PullTask 业务错误与连接生命周期解耦（P1/P2, TDD）
- [x] 24.1 Red: 新增测试，验证 `RequestTask` 的业务拒绝不会导致 runtime stream 断连。
- [x] 24.2 Red: 新增测试，验证兼容性拒绝场景下 heartbeat/config_update 链路仍保持可用。
- [x] 24.3 Green: 调整 `Connect` 的错误处理策略，将可预期业务拒绝改为“无任务/拒绝事件”而非流级错误返回。
- [x] 24.4 Refactor: 统一 gRPC 事件语义与 agent 侧退避策略，避免把业务拒绝计入连接故障。

## 25. 版本 tuple 预计算与调度热路径收敛（P1/P2, TDD）
- [x] 25.1 Red: 新增测试，验证创建期已校验配置后，调度期不再依赖重复 YAML 解析即可完成兼容性判定。
- [x] 25.2 Red: 新增测试，验证 tuple 来源在创建期与调度期一致且不可被后续配置文本漂移影响。
- [x] 25.3 Green: 在任务投影中持久化 `workflow/apiVersion/schemaVersion`（或等价 capability key），PullTask 直接读取。
- [x] 25.4 Refactor: 收敛 `extractWorkflowVersionTuple` 重复逻辑与版本正则定义，减少跨层重复解析。

## 26. `scan_task.config` 字段语义收敛（P2, TDD）
- [x] 26.1 Red: 新增测试，锁定任务创建时 `scan_task.config` 的期望行为（显式写入最小切片或保持空且禁用读取）。
- [x] 26.2 Red: 新增测试，验证 `task.Config` 与 `scan.YamlConfiguration` 不会形成双源歧义。
- [x] 26.3 Green: 在“按任务配置切片”与“删除冗余字段”两方案中选定其一并落地实现。
- [x] 26.4 Refactor: 更新 repository/domain 注释与 DTO 映射，清除半废弃字段语义。

## 27. `engineIDs`/`engineNames` 输入契约一致性（P2, TDD）
- [x] 27.1 Red: 新增测试，验证创建请求中 `engineIDs` 与 `engineNames` 存在冲突时必须拒绝。
- [x] 27.2 Red: 新增测试，验证持久化后的 `engineIDs`/`engineNames` 可双向还原同一引擎集合语义。
- [x] 27.3 Green: 增加输入一致性校验（含 ID->Name 映射验证）并收敛创建链路依赖源。
- [x] 27.4 Refactor: 更新 API 文档与错误提示，明确两字段的职责与优先级。

## 28. 任务配置传输介质解耦（P2/P3, TDD）
- [x] 28.1 Red: 新增测试，验证大配置场景不会因环境变量注入失败导致任务不可执行。
- [x] 28.2 Red: 新增测试，验证敏感配置不会通过容器环境变量链路暴露到非必要观测面。
- [x] 28.3 Green: 将配置传输从 `CONFIG` 环境变量迁移为文件挂载或受控引用通道。
- [x] 28.4 Refactor: 清理 env 注入路径与兼容逻辑，补充迁移窗口与回滚策略。
