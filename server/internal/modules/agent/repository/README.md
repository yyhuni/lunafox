# agent/repository

agent 模块 repository 规范：

- `agent.go`：`AgentRepository` 接口、仓储结构体与构造函数。
- `agent_query.go`：Agent 查询方法（`Find* / List`）。
- `agent_command.go`：Agent 写操作方法（`Create / Update / Delete / Update*`）。
- `registration_token.go`：注册令牌仓储接口与构造函数。
- `registration_token_query.go`：注册令牌查询方法。
- `registration_token_command.go`：注册令牌写操作与 never-attributed 条件清理。
- `agent_location.go`：Agent 最后成功位置快照仓储与构造函数。
- `agent_location_query.go`：Agent 位置快照查询。
- `agent_location_command.go`：按观测 IP 与 generation 栅栏执行完整快照替换或过期标记。
- `agent_cluster_summary_query.go`：在单条条件聚合查询中生成全集节点桶、15 秒容量新鲜度、守恒容量和位置覆盖事实。
- `agent_location_map_query.go`：不分页读取所有完整有效的 Agent 最后成功位置及地图所需最小运行态字段。

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。
- Agent 位置成功写必须在同一事务锁定 `agent_runtime_status` 并同时匹配 `observed_source_ip` 与 `observed_ip_generation`；旧 generation 不得改写或重新绑定既有 provenance。
- `RecordConnection` 在连接建立时原子写入当前 `connection_ip` 与其严格公网 `observed_source_ip` 投影；私网地址可展示但不能触发 GeoIP。断开和离线只清空 `connection_ip`，不抹除已有位置 provenance。
- Agent 注册必须在同一事务重检令牌尚未过期、首次写入 `ever_attributed_at` 并创建 Agent FK；Agent 创建失败必须回滚 marker。清理只删除已过 `expires_at + 24h`、marker 仍为空且 DELETE 当下不存在 Agent FK 的令牌。
- 注册令牌资源读取按 `registration_token.id` 返回所有当前关联 Agent，不应用 Agent collection 分页、搜索或其他令牌的行，也不得把 bearer secret 投影到读取模型。
- Agent 集群汇总必须使用调用方传入的同一个 `generatedAt`，严格按 `(generatedAt-15s, generatedAt]` 判定执行观测新鲜度；不得加载全集到应用内存或依赖 collection 分页。
- Agent 地图查询不得复用管理列表或 page token；离线 Agent 仍可返回 marker，缺失/无效位置必须在 repository 边界排除。
- Agent 管理列表 `List` 必须按 canonical 顺序执行：先 join `agent_runtime_status`，再应用白名单 `filter`，再应用 `createdAt + id` 稳定排序，最后分页。`totalSize` 必须统计筛选后的全集，不得只统计当前页。
- `/settings/agents/` 公开 contains 搜索依赖 PostgreSQL `pg_trgm`：`agent.display_name`、`agent_runtime_status.observed_hostname`、`agent_runtime_status.connection_ip` 必须在 baseline migration 中有 trigram 索引；`agent(created_at, id)`、`agent.status`、`agent_runtime_status(health_state, agent_id)` 是排序/筛选的审计依据。
