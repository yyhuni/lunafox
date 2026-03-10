## 1. Protocol foundation
- [x] 1.1 在 `proto/` 定义 AgentRuntime、AgentDataProxy、WorkerRuntime 三组 service 与消息 oneof
- [x] 1.2 配置 server/agent/worker 的 protobuf 代码生成流程（含 CI 校验）
- [x] 1.3 建立 gRPC 错误码与现有业务错误语义映射规范
- [x] 1.4 定义并落地 gRPC metadata 鉴权规范（`x-agent-key`、`x-worker-token`、task token），明确不引入 mTLS/SPIFFE

## 2. Release strategy (big-bang)
- [x] 2.1 明确一次性切换发布窗口、版本基线和回滚预案（整版回退）
- [x] 2.2 完成预发布全链路切换演练并记录检查清单
- [x] 2.3 锁定生产切换前的发布冻结与验证门禁
- [x] 2.4 完成 Nginx/LB 在统一 `443` 入口下对 HTTP 与 gRPC 上游端口的转发配置与验证

## 3. Server implementation
- [x] 3.1 新增 gRPC server 启动与路由注册（runtime stream + data proxy）
- [x] 3.2 实现 agent runtime 双向流处理：鉴权、心跳、任务请求与分配（pull-over-stream）、任务取消、状态上报
- [x] 3.3 实现 agent 数据代理接口：provider-config、wordlist、batch-upsert
- [x] 3.4 新增 server 侧 gRPC 单元/集成测试（含重连与异常路径）

## 4. Agent implementation
- [x] 4.1 用 gRPC 客户端替换 `agent/internal/websocket` 与任务 HTTP 客户端逻辑
- [x] 4.2 实现本地 WorkerRuntime UDS gRPC server，并完成 worker 请求代理
- [x] 4.3 调整 executor 与 worker 启动参数：注入 UDS 地址与 per-task token
- [x] 4.4 增加 agent 侧测试：流式事件处理、task lifecycle、worker proxy 鉴权
- [x] 4.5 实现 UDS socket 生命周期与权限：`lunafox_runtime` volume 创建、启动时清理旧 socket、关闭时主动删除、socket 权限设置（`0666`）、Worker 容器只读挂载

## 5. Worker implementation
- [x] 5.1 用 WorkerRuntime gRPC 客户端替换 `worker/internal/server` HTTP 客户端
- [x] 5.2 改写 provider-config/wordlist/batch-upsert 调用为本地 agent gRPC
- [x] 5.3 增加 worker 侧测试：UDS 连接、重试、错误码处理

## 6. Cutover and cleanup
- [x] 6.1 删除运行时旧接口与代码：`/api/agent/ws`、`/api/agent/tasks/*`、`/api/worker/*` 及对应客户端
- [x] 6.2 更新安装脚本、容器环境变量与运行文档
- [x] 6.3 执行端到端回归并确认管理面 HTTP API 无行为回归

## 7. Validation
- [x] 7.1 执行 `go test ./...`（server/agent/worker）
- [x] 7.2 执行关键集成测试（任务分发、取消、结果回传）
- [x] 7.3 执行 `openspec validate refactor-runtime-communication-to-grpc --strict --no-interactive`
