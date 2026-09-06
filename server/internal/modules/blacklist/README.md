# blacklist

`blacklist` 是 BlacklistPolicy 的唯一权威模块，拥有规则语言、规范化、etag、
global/Target-local singleton 的应用语义、持久化和 HTTP 映射。规则只支持精确
域名、`*.` 域名后代、IPv4 与 IPv4 CIDR；其他语法由 Server 拒绝，不能由前端
或 producer 自行扩展解释。

## 资源与持久化

- `blacklist_policy` 只有 `id`、`scope`、`target_id`、`patterns`、`updated_at`
  五列。etag 由规范化 `patterns` 的紧凑 JSON SHA-256 派生，不落库。
- 全局资源仅为 `blacklistPolicy`；Target-local 资源仅为
  `targets/{target}/blacklistPolicy`。两者只提供受保护的 GET/PATCH，PATCH 必须
  使用 `updateMask=patterns`、路径匹配的 `name`、非空 etag 与完整 local
  `patterns` 数组。
- Target 创建必须在同一事务内创建空 local Policy。Target-local Policy 不能
  删除或弱化 global Policy；有效并集只在 Scan 创建时由下游端口读取。

## Scan 与运行期边界

- Scan 生命周期拥有 `scan_blacklist_snapshot`，而非本模块。Scan repository 在
  每个 Scan 的创建事务内以 global 后 Target 的固定顺序共享锁读取 Policy，冻结
  规范化并集，并与 Scan、Task、saved plan 一起提交。之后的 Policy 修改只影响
  新 Scan。
- Server execution-input resolver 只读取该 immutable snapshot，并在 stream
  header 前编译 matcher。快照缺失、损坏或无法编译时必须 fail closed；不得回读
  当前 Policy 或将其当作空规则。
- 仅 Server 物化的 `subdomains`、`hostPorts`、`websiteURLs` 会排除命中 finalized
  facts。不得改写资产事实、`Execution.Target`、saved plan、Agent Context 或
  Engine/Agent 协议。Engine 自己生成的 Target baseline 不在此保证范围内。

## 可观测性

Policy GET/PATCH 只使用现有通用 HTTP request logging。本模块不新增 Policy
审计历史、安全事件、规则 payload/diff 日志、审计 API 或 UI。execution-input
过滤另有无敏感值的聚合终态日志和低基数指标，归属 agentdata 运行期边界。

## 验证

优先运行本模块的 domain/application/repository/handler/router 测试，再运行 Scan
快照、execution-input 和 PostgreSQL 并发测试。HTTP/前端边界调整还必须通过
`make verify-boundary-contracts`。
