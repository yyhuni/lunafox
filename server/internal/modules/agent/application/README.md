# agent/application

通用规则请遵循：`server/docs/server-application-naming-template-v1.md`

agent 模块补充规则：

- **入口聚合**：管理类业务入口单点收敛在 `facade_agent.go`。
- **服务编排**：`agent_query_service.go`、`agent_command_service.go`、`agent_registration_service.go` 承担 facade 背后的业务编排；`agent_control_lifecycle_service.go`、`agent_task_service.go` 作为 agent control-plane 链路的 transport-facing adapter service 保留，不替代模块级 facade。
- **control-plane 显式动作**：`agent_control_lifecycle_service.go` 在 lifecycle 边界显式暴露 `RecordHeartbeat`，registration snapshot 与后续 heartbeat 都先持久化并缓存 session epoch、node readiness、OS/arch、code-owned Engine API majors 和 distinct task counts；`agent_task_service.go` 只保留 container-execution `ClaimNextExecutionPlan` 以及包含必需 Engine diagnostics 的终态转发，终态只在 scan/workflow 同步处理成功后由 transport ack。
- **连接地址观测**：`connectionIp` 是 Server 在控制连接建立时写入的唯一当前展示观察，可为 Docker、NAT 或代理地址；控制连接断开或超时离线时只清空它。`observedSourceIp` 是其严格公网投影，只服务 GeoIP 与位置 provenance；两者都不参与认证、授权、调度或策略。
- **request_task 语义**：`RequestTask` 携带 session-scoped canonical request ID，只调用 saved-plan compatibility claim，不读取 package/config/workflow；响应使用 request-correlated `oneof{plan,no_task}`，普通 miss 返回显式 `no_task`，不存在 legacy assignment sibling 或执行 fallback。
- **freshness 与恢复分离**：scheduler capability freshness 使用短 heartbeat 窗口并对 missing/empty/stale/Docker-not-ready fail closed；同进程 session reconnect 的 fencing/recovery 窗口更长，不能把后者当作节点仍可领取任务的证据。
- **端口拆分**：端口使用资源化命名（如 `agent_query_ports.go`、`agent_command_ports.go`、`agent_store_ports.go`、`agent_message_ports.go`、`clock_ports.go`、`token_generator_ports.go`、`heartbeat_cache_ports.go`）。
- **Agent 管理列表查询**：`ListAgents` 只接受 `AgentListQueryInput{PageSize, PageToken, Filter, OrderBy}`，默认 `createdAt desc`，page token 绑定 normalized `filter/orderBy/pageSize`。公开筛选字段仅为 `displayName`、`observedHostname`、`connectionIp`、`status`、`healthState`；公开排序字段仅为 `createdAt`。不要恢复 `page`、独立 `status`、`include`、`sort*`、`keyword` 或旧 `ipAddress` 参数。
- **Agent 筛选项**：`ListFilterOptions` 仅开放 `status` 与 `healthState`，用于 `/settings/agents/` 后端来源筛选菜单。
- **注册令牌资源**：创建返回一次性 bearer `token` 与非密钥 `agentRegistrationTokens/{registration_token}` 资源名；后续状态读取只按数字资源 ID 返回完整归因 Agent 集合，不得返回 secret。never-attributed 清理由独立可取消后台 job 执行，不阻塞创建请求。
- **模型命名**：新增跨边界模型优先资源化命名（如 `*_item_models.go`、`*_query_inputs.go`）；默认实现保持在 `infrastructure` 并由 wiring 注入。
- **跨模块边界**：`agent_task_service.go` 只依赖 provider 模块暴露的 application bridge port，例如 `ScanTaskBridgePort`，不直接依赖对方内部 orchestration service。
- **历史迁移**：无历史聚合文件。
- **删除与 pinned Scan**：`AgentCommandService` 通过窄 `AgentDeletionLifecycle` 端口把删除交给 Scan repository 的同一数据库事务。删除前必须终止该 Agent 尚未开始的 pinned Scan task（`failure_kind=agent_deleted`）并收敛 Scan；不得清空 `scan.agent_id`、把任务重新入队或把 pinned Scan 降级为自动分配。运行中的任务仍由 session disconnect/fence 路径产生 `agent_disconnected`。
