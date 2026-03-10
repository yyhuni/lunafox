## 1. 状态语义与失败分类设计
- [x] 1.1 锁定少状态模型：`pending`、`running`、`completed`、`failed`、`cancelled`
- [x] 1.2 锁定 `failureKind` 字段语义和枚举清单
- [x] 1.3 明确 `running` 与 `failed` 的文档和监控口径

## 2. 仓储层与持久化（TDD）
- [x] 2.1 为任务失败新增或规范 `failureKind` 持久化字段
- [x] 2.2 为 claim 兜底失败、agent 断线、worker 启动失败、decode 失败增加失败测试
- [x] 2.3 保持 `pending -> running` 领取路径不变，并统一失败收口到 `failed + failureKind`

## 3. 应用层与 runtime（TDD）
- [x] 3.1 调整 `TaskRuntimeService` 失败路径，统一写入分类失败
- [x] 3.2 调整 gRPC runtime service 与 agent/runtime 错误映射，保证失败分类准确
- [x] 3.3 保留创建阶段严格校验，worker 解码失败作为兜底失败分类路径

## 4. 文档与观测收尾
- [x] 4.1 更新日志字段和监控说明，避免把所有 `failed` 都解释为“执行中失败”
- [x] 4.2 更新设计文档、OpenSpec delta 和测试基线

## 5. 验证
- [x] 5.1 运行相关 Go tests（scan / runtime / worker / agent）
- [x] 5.2 运行 `openspec validate refactor-scan-task-lease-state-machine --strict --no-interactive`
