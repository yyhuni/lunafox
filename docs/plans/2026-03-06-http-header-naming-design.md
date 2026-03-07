# HTTP Header Naming Modernization Design

## 背景
当前仓库同时存在三类 `X-*` HTTP 头：

1. 基础设施兼容头，例如 Nginx 转发使用的 `X-Forwarded-*`、`X-Real-IP`。
2. 应用观测头，例如 `X-Request-ID`。
3. 应用私有认证头，例如 `X-Agent-Key`、`X-Worker-Token`。

IETF 在 RFC 6648 中已经不推荐为新协议元素继续使用 `X-` 前缀，但现实世界里代理和网关层仍广泛保留历史兼容头。本仓库目前仍处于开发期，适合清理自定义业务协议命名，但不适合为了形式统一而重写代理兼容层。

## 目标
- 停止继续新增应用层 `X-*` 头。
- 在尚未形成稳定外部依赖前，收敛应用私有头命名。
- 保留基础设施兼容头，避免把代理层历史约定误当成业务协议问题。
- 为后续认证与链路追踪留出清晰演进路径。

## 非目标
- 不替换 Nginx / 代理层的 `X-Forwarded-*`、`X-Real-IP`。
- 不在本次设计里引入兼容双写或过渡桥接；默认按开发期一次性切换考虑。
- 不在本次设计里引入完整 Trace Context 体系；只为后续接入预留方向。

## 方案选项

### 方案 A：全部保持不变
优点：当前代码改动最少。

缺点：
- 继续积累 `X-*` 命名债务。
- 后续一旦真实接入 agent / worker 认证，会把旧命名扩散到更多地方。
- 无法区分“历史兼容头”和“新设计仍在造旧债”的边界。

### 方案 B：只治理应用私有头，保留基础设施兼容头（推荐）
优点：
- 改动集中，收益和风险比最好。
- 能在开发期一次性清理真正由项目自己定义的协议。
- 不会误伤代理 / 网关生态常见头。

缺点：
- 仓库中仍会保留一部分 `X-*`，视觉上不会完全消失。
- 需要明确哪些头属于“允许保留的兼容层”。

### 方案 C：全量去 `X-`
优点：表面上最统一。

缺点：
- 会把 `X-Forwarded-*` 这类事实标准也纳入重构，收益很低。
- 需要同时改代理配置、部署假设和调试经验，超出当前问题范围。
- 容易为了命名洁癖引入无意义 churn。

## 推荐方案
采用方案 B：只治理应用私有头，保留基础设施兼容头。

### 1. Request ID
- 当前：`X-Request-ID`
- 建议目标：`Request-Id`
- 适用文件：`server/internal/middleware/logger.go`

理由：
- 这是应用层请求关联标识，不属于必须依赖历史代理语义的头。
- 现在仍在开发期，一次性切换成本最低。
- 后续若要接入 `traceparent`，`Request-Id` 仍可作为本地排障辅助头保留。

补充约束：
- 代码常量统一使用一个名字，不做旧头兼容。
- 日志字段继续保留语义化写法，例如 `request.id` / `http.*`，不受 header 名改动影响。

### 2. Agent 认证头
- 当前：`X-Agent-Key`
- 建议目标：优先迁移到 `Authorization: Bearer <agent-token>`
- 适用文件：`server/internal/middleware/agent_auth.go`

理由：
- `Authorization` 不是 JWT 专用头，而是通用认证头。
- agent 路由目前是独立的 `/api/agent/*` 组，可以与用户 JWT 路由分开挂载认证逻辑，不会天然冲突。
- 使用标准认证头比继续保留私有 `X-*` 更利于后续文档、SDK、网关和审计工具接入。

补充约束：
- 若未来确实存在“同一条请求同时需要用户 JWT 和 agent 凭证”的场景，再单独设计第二认证通道。
- 在没有该场景前，不提前保留私有头。

### 3. Worker 认证头
- 当前：`X-Worker-Token`
- 建议目标：优先视为待清理遗留；若未来恢复 HTTP worker 入口，则默认使用 `Authorization: Bearer <worker-token>`
- 适用文件：`server/internal/middleware/worker_auth.go`

理由：
- 当前仓库说明已经明确：worker runtime 写入不再通过 HTTP `/api/worker/*`，而是走 gRPC runtime data proxy。
- 现有 `WorkerAuthMiddleware` 目前未挂到生产路由，继续围绕 `X-Worker-Token` 设计新协议收益很低。
- 更合理的顺序是先判定该中间件是否仍需保留，再决定未来 HTTP 认证写法。

补充约束：
- 如果短期确认没有 HTTP worker 入口恢复计划，应优先删除或标记 legacy，而不是重命名后继续保留。

### 4. 代理兼容头
- 保留：`X-Forwarded-For`、`X-Forwarded-Proto`、`X-Forwarded-Host`、`X-Forwarded-Port`、`X-Real-IP`
- 适用文件：`docker/nginx/nginx.conf`

理由：
- 它们属于基础设施兼容层，不是本仓库业务协议演进的主要问题。
- 即使标准上存在 `Forwarded`，现实部署里这批头仍是事实标准。

## 路由与认证边界

### 用户请求
- 继续使用 `Authorization: Bearer <jwt>`。
- 保持现有 `/api` 受保护路由分组不变。

### Agent 请求
- 在独立 `/api/agent/*` 路由组上定义 agent 认证。
- 默认使用 `Authorization: Bearer <agent-token>`，避免再引入新的私有 `X-*`。

### Worker 请求
- 当前不以 HTTP 入口为主，先按遗留清理处理。
- 若未来恢复 HTTP 路由，再单独评估是否与用户请求共享认证通道。

## 分阶段实施建议

### Phase 1：审计与定界
- 全仓扫描 `X-*` 使用点，标记为“保留 / 迁移 / 删除候选”。
- 明确 `worker_auth.go` 是否仍有实际产品路径依赖。
- 明确 agent 认证将挂载到哪些具体路由。

### Phase 2：测试先行
- 为 `logger` 增加基于新请求头名的测试。
- 为 `agent_auth` 增加基于 `Authorization` 的测试。
- 若保留 `worker_auth`，补足其未来目标行为测试；若删除，补回归测试确保无遗留引用。

### Phase 3：最小实现
- 将 `X-Request-ID` 常量切换为 `Request-Id`。
- 将 `AgentAuthMiddleware` 输入改为解析 `Authorization`。
- 对 `WorkerAuthMiddleware` 做删除或 legacy 隔离，不继续作为新协议入口扩展。

### Phase 4：文档与验证
- 更新安装文档、示例请求、测试样例和注释中的旧头名。
- 运行受影响 `go test`。
- 复查 Nginx / 代理配置，确认未误改基础设施兼容头。

## 风险与控制
- 风险：测试、脚本、示例请求仍使用旧头名。
  - 控制：在改常量前先全仓审计并更新测试。
- 风险：agent 未来出现双凭证同请求场景。
  - 控制：当前方案明确只在独立 agent 路由上复用 `Authorization`；若未来边界变化，再单独设计第二认证通道。
- 风险：误把基础设施兼容层纳入“去 `X-`”重构。
  - 控制：方案明确将 `docker/nginx/nginx.conf` 中相关头排除在迁移范围外。

## 决策摘要
- 现在适合改，但不适合“全量去 `X-`”。
- `X-Request-ID`：迁移到 `Request-Id`。
- `X-Agent-Key`：迁移到 `Authorization: Bearer ...`。
- `X-Worker-Token`：优先视为遗留清理对象；未来若恢复 HTTP 入口，默认也走 `Authorization`。
- `X-Forwarded-*` / `X-Real-IP`：保留。

## 下一步
如果认可本设计，下一步产出实施计划时应拆成三块独立任务：

1. `logger` 请求头迁移与测试。
2. `agent_auth` 认证头重构与路由挂载设计。
3. `worker_auth` 遗留清理或去留确认。
