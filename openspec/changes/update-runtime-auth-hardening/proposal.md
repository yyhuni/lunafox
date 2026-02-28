# Change: Runtime gRPC 鉴权与任务作用域加固

## Why
当前运行时链路已切换到 gRPC，但存在两个安全缺口：
- agent 本地 UDS 仅检查 `x-task-token` 非空，未绑定 `task_id`，存在任务越权风险。
- server 侧 AgentDataProxy 接口未强制校验调用方身份，存在未授权访问风险。

这些问题会导致跨任务数据读写和运行时数据面被滥用，需在不改变业务协议的前提下进行最小加固。

## What Changes
- 在 agent 本地 WorkerRuntime UDS 服务增加 `task_id <-> token` 作用域校验。
- task scope 校验失败时返回 `PermissionDenied`，与缺失/非法 token 的 `Unauthenticated` 区分。
- 在 server 侧 runtime gRPC server 增加统一鉴权拦截，强制 AgentDataProxy 仅接受带有效 `x-agent-key` 的请求。
- 为上述行为补充回归测试，覆盖失败路径与兼容路径。

## Impact
- Affected specs: `runtime-communication-grpc`
- Affected code:
  - `agent/internal/runtime/*`
  - `server/internal/grpc/runtime/server/*`
  - `server/internal/grpc/runtime/service/*`
