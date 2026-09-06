# agent/handler

agent 模块 handler 规范：

- handler 层仅处理 HTTP 绑定、错误映射与响应组装，不承载业务编排。
- 主资源 canonical collection list handler 使用 `List`，不要在 `AgentHandler` receiver 已表达资源时重复写 `ListAgents`。
- `GET /v1/admin/agents` 是已迁移 canonical collection list：只接受 `pageSize`、`pageToken`、`filter`、`orderBy`，并 hard cut 拒绝 `page`、独立 `status`、`include`、`sort`、`sortBy`、`sortOrder`、`keyword`。列表响应默认包含页面卡片所需 heartbeat projection；不要恢复 `include=heartbeat` 列表开关。
- `GET /v1/admin/agents/filterOptions?field=status|healthState` 返回后端聚合筛选项；路由必须注册在 `/:agent` 前面。
- `GET /v1/admin/agentClusterSummaries/current` 返回覆盖完整 Agent identity 集合的 fixed read-only summary；Overview 与 Agent 管理页不得从分页 collection 重建全局总数、容量或集群结论。
- `GET /v1/admin/agentLocationMaps/current` 返回完整、非分页的 current/expired Agent 位置投影和 nullable Server 最后成功快照；地图不得遍历 Agent collection、采样 marker 或补默认坐标。
- `GET /v1/admin/agents/:agent` 直接返回观测来源 generation 与最后成功位置状态；详情不得暴露 GeoIP attempt/failure 诊断，也不得把旧位置 provenance 改写为当前观测 IP。
- Agent 注册令牌管理使用 `POST /v1/admin/agentRegistrationTokens` 与 `GET /v1/admin/agentRegistrationTokens/:registrationToken`；旧 `/v1/admin/agents/registrationTokens` 创建路径已 hard cut。路径参数是非密钥数字资源 ID，不能使用 bearer secret。
- Agent 安装脚本统一使用 `GET /v1/agents:downloadInstallScript?registrationToken=...&profile=...` custom method。
- `profile` 只接受 `internal` / `external`：`internal` 使用内部 Agent control URL 并要求部署 Docker network；`external` 使用 `PUBLIC_URL` 且 network 可选。
- 旧 `local` / `remote` profile 名称不得作为兼容输入或新文档示例。
- Agent 安装脚本通过 `docker plugin ls/enable/install` 管理宿主 `loki` logging plugin；不得使用不存在的 `docker engine` 子命令，也不得在插件不可用时静默降级启动。
- `TestAgentInstallScriptRegistersAndConnects` 是 Linux Docker 的 opt-in 端到端冒烟测试：它下载并执行真实安装脚本、启动正式 Agent 镜像，并等待注册、control-plane session 和首个 heartbeat。独立的 `Docker Runtime E2E` workflow 会在相关 Agent/安装/契约/Docker 路径变更的 pull request 中运行 `LUNAFOX_RUN_AGENT_INSTALL_E2E=1 go test -count=1 ./internal/modules/agent/handler -run '^TestAgentInstallScriptRegistersAndConnects$' -v`，并保留 `workflow_dispatch` 供人工复跑；它不会仅因 pull request 合并后的分支 push 再重复运行。受保护分支应将该 workflow check 设为 required。它使用固定的 Agent 容器和 volume 名称，发现已有同名资源时必须 fail-fast。
- `GET /v1/admin/agents/:agent/logEntries` 是 Agent log viewer 的 operational contract：它返回 `nextPageToken`、`previousPageToken`、`hasOlder`、`hasNewer`、`caughtUp`、`gap` 等查看器元数据，用于 history hydration 与 live follow；当前阶段仍基于短周期拉取，不承诺 push streaming。`pageSize` 未传时为 200，必须为 `1..500`；`100/200/500/1000/2000/5000` 是前端 latest-N 窗口，后端每次只立即返回一页且不跨请求为 viewer 累积结果。
