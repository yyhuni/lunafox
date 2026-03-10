## Context
当前系统的运行时链路由多协议拼接而成，协议边界与认证边界不一致：
- Agent 通过 WebSocket 上报心跳并接收控制消息，同时通过 HTTP 拉任务和回写状态。
- Worker 直接调用 Server 的 HTTP API 获取配置、下载字典、上报结果。
- Agent 与 Worker 缺少标准化 RPC 边界，协作主要依赖容器环境变量与退出码。

目标是将运行时内部通信统一为 gRPC，形成清晰的控制面与数据面。

## Goals / Non-Goals
- Goals:
  - 统一内部运行时通信协议为 gRPC。
  - 建立 `worker -> agent -> server` 的单向信任链，消除 worker 直连 server。
  - 用 protobuf 统一消息类型与版本管理。
  - 为任务生命周期与结果写入提供流式/类型化能力。
- Non-Goals:
  - 不改动 Web 管理台对外 HTTP/JSON API。
  - 不在本次引入跨机 mTLS 证书体系，先沿用现有 token 认证语义映射到 gRPC metadata。
  - 不在本次新增 gRPC 可观测性基础设施（OpenTelemetry、Prometheus exporter、trace propagation）。统一协议后可观测性基础已具备，instrumentation 作为后续独立 change 处理。但保留最小可观测性基线（见 Decisions）。

## Approaches Considered
- Approach A: 仅将 `worker <-> server` 改为 gRPC，其他保持现状。
  - Pros: 改造面最小，见效快。
  - Cons: 仍有 WebSocket + HTTP + gRPC 三套链路，不满足统一目标。

- Approach B: `agent <-> server` 与 `worker <-> server` 都改 gRPC，worker 继续直连 server。
  - Pros: 实现路径较直。
  - Cons: worker 仍持有 server 凭证，安全边界不清晰；agent 价值被削弱。

- Approach C (Recommended): 统一为 `worker <-> agent`(UDS gRPC) + `agent <-> server`(gRPC)。
  - Pros: 协议与信任边界最清晰；后续扩展和观测链路一致。
  - Cons: agent 需要承担代理职责，初期改造量最大。

## Decisions
- Decision: 采用 Approach C。
  - `agent <-> server` 使用 gRPC 双向流作为控制通道。
  - `worker <-> agent` 使用 UDS gRPC 作为本机数据通道。

- Decision: 控制面采用单条长连接双向流。
  - Agent 上行消息：heartbeat、request_task、task_status。
  - Server 下行消息：task_assign（作为 request_task 的响应）、task_cancel、config_update、update_required。
  - 不设独立 ack 消息：Agent 收到 `task_assign` 后直接上报 `task_status(running)` 作为隐式确认。Pull-over-Stream 模型下 Agent 不存在拒绝场景，独立 ack 是多余的握手。

- Decision: 任务分发保留 Pull 语义（Pull-over-Stream），不采用 Server 主动推送模式。
  - Agent 在有空余容量时发送 `request_task`，Server 收到后从队列取一个任务以 `task_assign` 响应；无可用任务时 Server 持有请求等待。
  - 背压天然解决：Agent 侧 `canPull()` 逻辑（maxTasks、CPU/内存/磁盘阈值）完整保留，仅在通过时才发 `request_task`。
  - 相比 push 模式的优势：不需要 Server 维护 per-agent 容量状态；不需要 nack 机制和拒绝后重入队；现有 Puller 代码迁移成本最低。
  - 相比现有 HTTP 轮询的优势：无轮询开销，request 在流上等待、Server 一有任务立即响应，延迟接近零。
  - `task_cancel`、`config_update`、`update_required` 仍由 Server 主动推送（这些不需要背压）。

- Decision: server gRPC 使用独立内网监听端口，不复用现有 HTTP 端口。
  - 外部入口继续使用现有 Nginx `443`，由反向代理将 gRPC（HTTP/2）转发到 gRPC listener。
  - 管理面 HTTP/JSON 继续转发到现有 HTTP listener。

- Decision: 数据面由 agent 代理。
  - Worker 请求 provider config、wordlist、结果写入都发往本机 agent。
  - Agent 调用 server gRPC 数据服务完成转发与落库。

- Decision: Worker-Agent UDS 拓扑采用单一共享 socket + per-task token 鉴权。
  - Agent 整个生命周期维护一个 UDS gRPC server，所有 Worker 共用。
  - 每个任务分配独立的 session token，Agent 在请求级别校验 token 与 taskId 的绑定关系。
  - 不采用 per-task 独立 socket 方案：当前 Worker 执行的是自有构建的扫描镜像（非用户上传代码），UDS + per-task token 已提供充分隔离；per-task socket 增加的生命周期管理（N 个 listener/FD/goroutine、异常退出清理、运维排障成本）在当前阶段收益不成比例。
  - 后续演进：若引入用户自定义 Worker 插件或强多租户合规要求，可升级为 per-task socket 方案，Worker 侧代码无需改动（仅感知 `AGENT_SOCKET` 环境变量）。

- Decision: UDS 通过独立 Docker named volume 共享，不复用现有数据卷。
  - 新增 `lunafox_runtime` named volume，与现有 `lunafox_data`（持久化数据卷）职责分离。
  - Agent 以读写模式挂载，Worker 以只读模式挂载。
  - 不使用宿主机路径绑定，与现有 named volume 模式保持一致，避免跨平台兼容问题。

- Decision: 旧运行时 HTTP/WS 端点在 cutover 后直接删除，不做长期双栈。

- Decision: 保留最小可观测性基线（日志级别，不引入外部依赖）。
  - 流连接数（gauge）：Agent 连接/断开时记录日志。
  - 重连次数（counter）：每次重连尝试记录日志，含退避时长。
  - 鉴权失败数（counter）：Agent 侧和 UDS 侧 interceptor 拒绝时记录日志，含 token 指纹（hash/前缀）与来源 IP；禁止记录原始 token。
  - 代理延迟（per-request）：Agent 代理 Worker 请求到 Server 时记录耗时（debug 级别日志）。
  - 实现方式：纯日志输出（`zap.Info`/`zap.Debug`），不引入 Prometheus client 或 OTel SDK。

- Decision: 迁移采用一次性切换，不提供临时双栈兼容模式。
  - 仅允许在预发布环境进行并行验证。
  - 生产切换时要求 server/agent/worker 版本同步。

## Proposed Proto Surface

### Service 定义

```protobuf
// 控制面：Agent ↔ Server 双向流
service AgentRuntimeService {
  rpc Connect(stream AgentRuntimeRequest) returns (stream AgentRuntimeEvent);
}

// 数据面：Agent → Server（Agent 代理 Worker 请求）
service AgentDataProxyService {
  rpc GetProviderConfig(GetProviderConfigRequest) returns (GetProviderConfigResponse);
  rpc GetWordlistMeta(GetWordlistMetaRequest) returns (GetWordlistMetaResponse);
  rpc DownloadWordlist(DownloadWordlistRequest) returns (stream DownloadWordlistChunk);
  rpc BatchUpsertAssets(BatchUpsertAssetsRequest) returns (BatchUpsertAssetsResponse);
}

// 数据面：Worker → Agent（本机 UDS）
service WorkerRuntimeService {
  rpc GetProviderConfig(GetProviderConfigRequest) returns (GetProviderConfigResponse);
  rpc EnsureWordlist(EnsureWordlistRequest) returns (EnsureWordlistResponse);
  rpc PostBatch(PostBatchRequest) returns (PostBatchResponse);
}
```

### Agent 上行消息（AgentRuntimeRequest oneof）

```protobuf
message AgentRuntimeRequest {
  oneof payload {
    Heartbeat     heartbeat     = 1;
    RequestTask   request_task  = 2;
    TaskStatus    task_status   = 3;
  }
}

message Heartbeat {
  double cpu_usage       = 1;
  double mem_usage       = 2;
  double disk_usage      = 3;
  int32  running_tasks   = 4;
  string version         = 5;
  string hostname        = 6;
  int64  uptime_seconds  = 7;
  HealthStatus health    = 8;
}

message HealthStatus {
  string state   = 1; // "healthy" | "degraded" | "unhealthy"
  string reason  = 2;
  string message = 3;
}

message RequestTask {
  // 空消息。Agent 仅在 canPull() 通过后发送，无需携带容量信息。
}

message TaskStatus {
  int32  task_id  = 1;
  string status   = 2; // "running"（隐式 ack）| "completed" | "failed" | "cancelled"
  string message  = 3; // 失败原因或取消原因（可选）
  int32  exit_code = 4; // Worker 退出码（completed/failed 时有值）
}
```

### Server 下行消息（AgentRuntimeEvent oneof）

```protobuf
message AgentRuntimeEvent {
  oneof payload {
    TaskAssign      task_assign      = 1;
    TaskCancel      task_cancel      = 2;
    ConfigUpdate    config_update    = 3;
    UpdateRequired  update_required  = 4;
  }
}

message TaskAssign {
  // 作为 RequestTask 的响应。found=false 表示当前无可用任务。
  bool   found          = 1;
  int32  task_id        = 2;
  int32  scan_id        = 3;
  int32  stage          = 4;
  string workflow_name  = 5;
  int32  target_id      = 6;
  string target_name    = 7;
  string target_type    = 8;
  string workspace_dir  = 9;
  string config         = 10; // JSON 扩展配置
}

message TaskCancel {
  int32 task_id = 1;
}

message ConfigUpdate {
  optional int32 max_tasks       = 1;
  optional int32 cpu_threshold   = 2;
  optional int32 mem_threshold   = 3;
  optional int32 disk_threshold  = 4;
}

message UpdateRequired {
  string target_version = 1;
  string image_ref      = 2;
}
```

### request_task 长持有语义

Agent 发出 `request_task` 后，如果 Server 没有可用任务，Server 不会立即返回空响应，而是 **持有该请求等待**，直到以下任一条件触发：

- **有可用任务**：Server 以 `TaskAssign{found=true, ...}` 响应。
- **持有超时（60s）**：Server 以 `TaskAssign{found=false}` 响应，Agent 收到后重新检查 `canPull()` 再决定是否重发。
- **流关闭**：Agent 关闭或断连时，流关闭会自动取消所有悬挂的请求，无需额外处理。

设计要点：
- 超时设为 60s 而非无限持有：防止 Server 端积累过多 goroutine；Agent 定期重新评估系统状态（CPU/内存可能变化）。
- Agent 不维护 in-flight `request_task` 状态：发出后等待响应即可，响应到达后再决定下一步。控制流完全串行：`canPull() → request_task → 等待 → task_assign → task_status(running) → 处理完成 → task_status(completed/failed) → 回到 canPull()`。
- Agent 关闭时无需特殊取消逻辑：context 取消 → 流关闭 → Server 自动清理持有的 request。


## Worker-Agent UDS Design

### Socket 路径与 Volume 策略

| 资源 | 值 |
|------|----|
| Named volume | `lunafox_runtime` |
| Agent 挂载 | `lunafox_runtime:/run/lunafox` (rw) |
| Worker 挂载 | `lunafox_runtime:/run/lunafox:ro` |
| Socket 路径（容器内） | `/run/lunafox/worker-runtime.sock` |

路径选择 `/run/lunafox/` 而非 `/var/run/lunafox/`：`/run` 是 Linux FHS 标准的临时运行时目录，语义更准确。

独立 volume 而非复用 `lunafox_data` 的原因：
- `lunafox_data` 是持久化数据卷（wordlists、workspace），Server 容器也挂载了它。
- Socket 是临时运行时文件，不应混入持久卷，也不应暴露给 Server 容器。

### 生命周期管理

```
Agent 启动
├── os.Remove("/run/lunafox/worker-runtime.sock")  // 清理可能残留的旧 socket
├── net.Listen("unix", "/run/lunafox/worker-runtime.sock")
├── 启动 WorkerRuntimeService gRPC server
│
│  收到 task_assign:
│  ├── token := crypto.GenerateSessionToken()
│  ├── workerServer.RegisterTask(token, TaskContext{taskID, scanID, targetID})
│  ├── docker.StartWorker(ctx, task, token)  // 注入 TASK_TOKEN 和 AGENT_SOCKET
│  ├── docker.Wait(...)                      // 阻塞等待 Worker 退出
│  └── workerServer.UnregisterTask(token)    // defer，无论退出原因都执行
│
Agent 关闭
├── grpcServer.GracefulStop()  // 等待进行中的请求完成
├── listener.Close()
└── os.Remove(socketPath)      // 主动清理 socket 文件
```

关键行为：
- Agent 启动时无条件清理旧 socket（覆盖 Agent 崩溃后残留场景）。
- 每个任务的 token 在 `execute()` 的 defer 中注销，覆盖所有退出路径（正常、失败、超时、取消）。
- Socket 权限设为 `0666`：Worker 容器以 root 运行，volume 为只读挂载（Worker 不可创建新文件，但可读写已有 socket）。

### 环境变量规划

| 变量 | 角色 | 设置者 | 消费者 |
|------|------|--------|--------|
| `LUNAFOX_RUNTIME_VOLUME` | runtime volume 名称（默认 `lunafox_runtime`） | install script | Agent |
| `LUNAFOX_RUNTIME_SOCKET` | socket 完整路径（默认 `/run/lunafox/worker-runtime.sock`） | Agent 配置 | Agent（listen） |
| `AGENT_SOCKET` | Worker 端使用的 socket 路径 | Agent（注入 Worker env） | Worker（dial） |
| `TASK_TOKEN` | Per-task 鉴权 token | Agent（注入 Worker env） | Worker（gRPC metadata） |

移除的环境变量（Worker 不再需要）：
- `SERVER_URL` — Worker 不再直连 Server。
- `SERVER_TOKEN` — Worker 不再持有 Server 凭证。

### Docker 挂载变更

Agent 容器（install script）：
```bash
$DOCKER_CMD run -d --restart unless-stopped --name lunafox-agent \
  # ... 现有参数 ...
  -v "$SHARED_DATA_VOLUME_BIND" \
  -v "${RUNTIME_VOLUME}:/run/lunafox" \          # 新增
  -e LUNAFOX_RUNTIME_SOCKET=/run/lunafox/worker-runtime.sock \  # 新增
  "$AGENT_IMAGE_REF"
```

Worker 容器（Agent 通过 Docker API 创建）：
```go
hostConfig := &container.HostConfig{
    Binds: []string{
        sharedDataVolumeBind,                                      // 已有
        fmt.Sprintf("%s:/run/lunafox:ro", runtimeVolumeName),      // 新增
    },
}
env := []string{
    "AGENT_SOCKET=/run/lunafox/worker-runtime.sock",
    fmt.Sprintf("TASK_TOKEN=%s", taskToken),
    // ... 其他任务参数（SCAN_ID, TARGET_ID 等保留）
}
```

### 容器路径总览

```
容器        路径                                  来源                  模式
──────────  ────────────────────────────────────  ────────────────────  ────
Agent       /opt/lunafox/                         lunafox_data          rw
Agent       /run/lunafox/worker-runtime.sock      lunafox_runtime       rw
Agent       /var/run/docker.sock                  宿主机 docker.sock    rw

Worker      /opt/lunafox/                         lunafox_data          rw
Worker      /run/lunafox/worker-runtime.sock      lunafox_runtime       ro
```

### 异常与清理策略

| 场景 | 处理 |
|------|------|
| Worker 崩溃 | Agent 通过 `docker.Wait()` 检测退出 → `defer UnregisterTask(token)` 注销 token |
| Agent 正常关闭 | `GracefulStop()` → `listener.Close()` → `os.Remove(socketPath)` 主动清理 |
| Agent 异常退出（OOM/panic） | Socket 文件残留在 named volume → 下次启动时 `os.Remove` 无条件清理 |
| Agent 重启/断连期间 Worker | Worker 数据面请求失败 → 指数退避重试 3 次（~15s）→ 耗尽后 graceful exit（详见 [Stream Reconnection Strategy](#stream-reconnection-strategy)）|
| 无效 token 请求 | gRPC interceptor 拒绝，返回 `codes.Unauthenticated`，请求不转发到 Server |
| 并发 Worker | 单一 gRPC server 天然支持并发客户端，goroutine-per-request 模型 |

**Volume 生命周期（安装脚本管理）：**
- `lunafox_runtime` volume 跟随 Agent 容器长期存在，运行期间只含一个 socket 文件，体积可忽略。
- 安装脚本在 Agent 卸载或重装时负责清理：
  ```bash
  $DOCKER_CMD rm -f lunafox-agent
  $DOCKER_CMD volume rm lunafox_runtime
  ```
- 与 `lunafox_data` 的 volume 管理方式保持一致。

## Auth and Security
- 本阶段认证策略:
  - 明确保留现有 token 语义，通过 gRPC metadata 传递与校验。
  - 本次变更不引入 mTLS/SPIFFE，不新增证书签发/轮换基础设施。
- 后续演进策略:
  - 证书强身份（mTLS/SPIFFE）作为后续独立 change 处理，避免与本次一次性切换耦合。

- `agent <-> server`:
  - 在 gRPC metadata 传递 `x-agent-key`，保持现有 agent 身份语义。
  - `x-worker-token` 仅用于 agent 代表 worker 调用数据面接口时的服务间授权，避免 worker 暴露 server token。
- `worker <-> agent`:
  - 传输层隔离：Unix Domain Socket，Worker 容器只读挂载，无需网络端口暴露。
  - 应用层鉴权：Agent 为每个任务生成 per-task session token，通过 gRPC metadata `x-task-token` 传递。
  - Agent 侧 gRPC unary interceptor 校验 token 有效性，并验证 token 绑定的 taskId/scanId 与请求参数一致，防止越权。
  - Token 生命周期与任务绑定：`RegisterTask` 在任务启动前、`UnregisterTask` 在任务结束后（defer），过期 token 自动拒绝。

## Stream Reconnection Strategy

### Agent 侧重连退避

- 指数退避：base=1s, max=30s, 每次翻倍。
- 添加 ±20% 随机 jitter 防止多 Agent 同时重连（thundering herd）。
- 连接成功后立即 reset 退避。
- 参数相比现有 WebSocket（base=1s, max=60s, 无 jitter）更紧凑：gRPC 连接建立更轻量，30s 上限足够。

### 重连后 Agent 行为（干净重置）

- Agent 视每次重连为"新连接"，不尝试补发断连期间积累的 `task_status`。
- 重连后立即发送 heartbeat，按正常流程判断 `canPull()` 后决定是否发 `request_task`。
- 不补发的理由：
  - Server 的 `AgentMonitor` 已有兜底机制（心跳超时 → offline → `FailTasksForOfflineAgent`）。
  - 补发引入时序问题：AgentMonitor 可能已将任务标记为 failed，补发 completed 需要额外的状态优先级判断，复杂度不值得。
  - 断连窗口通常很短（秒级），仍在运行的 Worker 在 Agent 重连后会正常触发状态上报。

### 断连期间 Worker 处理

- Agent 与 Server 断连时，Worker → Agent UDS 通信不受影响。
- Agent 无法转发数据面请求到 Server 时，对所有 Worker 数据面请求返回 `codes.Unavailable`。
- Worker gRPC 客户端收到 `codes.Unavailable` 后按自身重试策略处理（指数退避 3 次，总窗口 ~15s）。
- 重试耗尽后 Worker 以非零退出码退出，Agent 检测到退出后根据任务状态决定是否重新调度。
- 不在 Agent 侧缓冲 Worker 请求：避免引入内存队列管理、有界队列溢出、断连恢复后重放顺序等复杂性。

### Server 侧 Agent 离线检测

- **即时信号**：gRPC 双向流关闭时，Server handler 的 `defer` 触发 `OnDisconnected(agentID)` → 标记 Agent offline。
- **兜底机制**：保留现有 `AgentMonitor` 不变 — 定期扫描心跳超时的 Agent → 标记 offline → `FailTasksForOfflineAgent`。
- 双层保障：流关闭提供秒级即时检测，AgentMonitor 覆盖网络半开（流未关闭但心跳已停）的边缘场景。

## Migration Plan
1. 建立 proto 契约与代码生成链路，提交 server/agent/worker stub。
2. 实现 server 端 gRPC runtime 服务与数据代理服务。
3. 实现 agent gRPC 客户端（到 server）和本地 gRPC 服务端（给 worker）。
4. 实现 worker gRPC 客户端，改写现有 `worker/internal/server` HTTP 客户端调用点。
5. 在预发布完成一次性切换演练（不启用生产双栈）。
6. 更新 Nginx/LB 转发规则：统一 `443` 入口下区分 HTTP 与 gRPC 上游端口。
7. 生产窗口内同步发布 server/agent/worker 并切换到 gRPC。
8. 删除旧运行时 WebSocket/HTTP 路由与客户端逻辑。
9. 完成端到端联调与回归测试，更新安装脚本与运维文档。

## Risks / Trade-offs
- 风险: UDS 路径与容器挂载配置出错会导致 worker 启动即失败。
  - Mitigation: agent 在启动 worker 前进行 socket readiness 检查并输出明确错误。

- 风险: 字典下载改为 gRPC 流后，超时与重试策略与当前 HTTP 不同。
  - Mitigation: 定义统一分块大小、超时、重试上限，并补充断点恢复策略（后续增强）。

- 风险: 一次性切换失败将影响全量运行时任务。
  - Mitigation: 预先执行全链路演练、冻结发布窗口、准备快速回滚（回滚为整版回退，不回退到双栈）。

## Validation Strategy
- 单元测试:
  - proto mapping、状态机转换、鉴权元数据解析。
- 集成测试:
  - `agent <-> server` 双向流重连、任务下发、取消、状态上报。
  - `worker <-> agent` UDS 调用与失败场景。
- 端到端测试:
  - 创建扫描任务到结果入库全链路。
- 回归:
  - 管理面 Web API 与页面行为不受影响。
