# Runtime 旧 Envelope 收敛说明

## 背景
历史上 server/agent 侧存在基于自定义 envelope 的消息模型（如 `agentproto.Message`、`agent/internal/protocol`）。
当前 runtime 主链路已迁移到 gRPC 双向流协议（`contracts/gen/lunafox/runtime/v1`）。

## 当前策略
- runtime 主链路（心跳、拉任务、下行事件）以 gRPC proto 为单一来源。
- 历史 envelope 类型保留为 `Deprecated`，仅用于兼容与局部测试夹具，不再作为主链路输入输出契约。

## 开发约束
- 新增 runtime 行为必须直接修改 `proto/lunafox/runtime/v1/runtime.proto` 与对应 mapper。
- 不得在 runtime 主链路代码中重新引入 `agentproto.Message`/`agent/internal/protocol` 依赖。

## 迁移边界
- 兼容窗口内允许保留历史类型定义，但禁止扩展其业务能力。
- 后续清理阶段可直接删除历史 envelope 类型及其桥接层。

## RequestTask 事件语义（业务拒绝 vs 连接故障）
- `RequestTask` 业务拒绝（如兼容性不满足、配置校验失败）不会再中断 runtime stream。
- server 会返回 `task_assign{found:false}`，表示“当前无可分配任务/业务拒绝已处理”。
- agent 侧将 `found:false` 视为“空任务”路径，走 empty backoff，不计入连接故障重连退避。

这条约束用于避免把业务拒绝误判为传输层故障，防止无意义重连抖动影响心跳和下行事件稳定性。
