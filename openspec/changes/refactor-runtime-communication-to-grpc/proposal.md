# Change: 运行时通信链路统一为 gRPC

## Why
当前运行时通信存在三套机制并行：
- `agent <-> server` 使用 WebSocket + HTTP。
- `worker <-> server` 使用 HTTP + 静态 token。
- `worker <-> agent` 仅通过容器启动参数和退出码间接协作，没有标准 RPC。

这会导致协议语义分散、认证模型不一致、类型边界重复定义、端到端可观测性断裂。项目当前仍处于开发阶段，适合一次性收敛到统一通信栈，降低后续演进成本。

## What Changes
- 新增运行时 gRPC 协议层（protobuf）并统一服务契约。
- `agent <-> server` 改为单条双向流控制通道（心跳、任务请求与分配、任务取消、配置更新、状态回传）。
- `worker <-> agent` 新增本机 Unix Domain Socket gRPC 通道，worker 不再直连 server。
- `agent <-> server` 新增数据平面 gRPC 调用，由 agent 代理 worker 的配置读取、字典下载、结果写入请求。
- 鉴权策略沿用现有 token 体系并映射到 gRPC metadata，本次不引入 mTLS/SPIFFE。
- 部署形态采用独立 gRPC 内网监听端口（不与现有 HTTP listener 复用端口），并通过现有 Nginx `443` 统一入口转发。
- 迁移策略采用一次性切换（big-bang cutover），不保留运行时双栈兼容窗口。
- 移除旧运行时链路：
  - `/api/agent/ws`
  - `/api/agent/tasks/pull`
  - `/api/agent/tasks/:taskId/status`
  - `/api/worker/*` 运行时接口
- 外部管理面接口（Web 控制台使用的 HTTP/JSON API）保持不变。

## Impact
- Affected specs: `runtime-communication-grpc` (new capability delta)
- Affected code:
  - `server/internal/modules/agent/*`
  - `server/internal/modules/scan/*`
  - `server/internal/modules/catalog/*`
  - `server/internal/middleware/*`
  - `server/internal/bootstrap/*`
  - `server/internal/websocket/*` (to be removed/replaced)
  - `agent/internal/websocket/*` (to be removed/replaced)
  - `agent/internal/task/*`
  - `agent/internal/docker/*`
  - `worker/internal/server/*` (to be removed/replaced)
  - `worker/internal/config/*`
  - 新增 `proto/` 与代码生成产物目录
- Deployment impact:
  - 发布窗口内要求 server/agent/worker 同步升级。
  - 旧运行时客户端在切换后将不可继续工作。
