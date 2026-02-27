## Context
当前安装脚本通过单一路由 + `mode=local|remote` 选择端点，带来两个核心问题：
- 用户需要理解分支语义，且若改成手填端点变量会显著增加操作负担。
- 远程 VPS 场景下，若运行时端点误用 `http://server`，跨机网络不可达会直接失败。

本次目标是在不要求用户手动填写运行期端点变量的前提下，同时稳定支持本地同网部署与远程 VPS 部署。

## Goals / Non-Goals
- Goals:
  - 用户零手填运行期端点变量（`REGISTER_URL`、`RUNTIME_GRPC_URL`）。
  - 入口显式化（本机安装 / 远程安装），移除隐式 `mode` 分支与兼容回退。
  - 本地与远程场景均可达，错误行为 fail-fast。
- Non-Goals:
  - 不变更 runtime gRPC 协议、鉴权模型与 agent-server 业务语义。
  - 不改动现有 nginx gRPC 反代基础能力。
  - 不在本次引入新的证书体系（mTLS/SPIFFE）。

## Multi-Agent 会审结论

### 子代理A（产品体验）
- 结论：必须“零手填”。
- 理由：让用户输入 URL 在安装阶段出错率高，且用户难以区分注册端点和运行时端点。
- 建议：提供两个明确安装入口，由服务端预渲染必须变量。

### 子代理B（网络可达性）
- 结论：不能对所有场景固定 `http://server`。
- 理由：`server` 仅在同 Docker network 有效，远程 VPS agent 不可解析。
- 建议：`local` 使用内网端点，`remote` 使用公网可达端点。

### 子代理C（工程维护）
- 结论：删除 `mode` query，避免隐式分支与默认回退。
- 理由：mode 语义不透明，历史兼容分支持续增加测试与排障成本。
- 建议：通过显式路由表达 profile，`mode` 参数传入即 400。

## Approaches Considered

### Approach A: 全量单入口 + 全走 PUBLIC_URL
- Pros:
  - 代码最少，路径最短。
- Cons:
  - 本地部署依赖公网回环与证书环境，调试链路不友好。

### Approach B（Recommended）: 双入口预设 + 零手填
- 方案：
  - 新增两个入口：
    - `/api/agent/install-script/local`
    - `/api/agent/install-script/remote`
  - 服务端按入口预渲染：
    - local: `REGISTER_URL=PUBLIC_URL`, `RUNTIME_GRPC_URL=http://server:<grpc-port>`
    - remote: 两者均为 `PUBLIC_URL`
  - 拒绝 `mode` 参数，不做回退。
- Pros:
  - 用户体验简单（只选“本机/远程”）。
  - 远程与本地均可达，拓扑语义明确。
- Cons:
  - 需要新增路由与文档迁移工作。

### Approach C: 保持单入口并让用户手填端点变量
- Pros:
  - API 结构改动小。
- Cons:
  - 用户心智负担和出错率最高，不符合“安装简单化”目标。

## Decisions
- Decision: 采用 Approach B（双入口预设 + 零手填）。
- Decision: `mode` 参数删除语义为“传入即 400”，且不做任何 fallback。
- Decision: local profile 中 Docker network 缺失时失败退出，不降级到 default bridge。
- Decision: 保持 `PUBLIC_URL` 的现有 https 校验逻辑。

## Ownership Contract
- 项目安装器链路（`tools/installer`）必须固定调用 `local` 入口，不得调用 `remote` 入口。
- 前端安装弹窗链路必须固定调用 `remote` 入口，不得调用 `local` 入口。
- 后端不得再通过 query 参数推断 profile，profile 仅由路由表达。
- 以上归属必须由自动化测试覆盖，防止回归。

## Endpoint Matrix
- local profile:
  - `REGISTER_URL = PUBLIC_URL`
  - `RUNTIME_GRPC_URL = http://server:<SERVER_GRPC_PORT>`
- remote profile:
  - `REGISTER_URL = PUBLIC_URL`
  - `RUNTIME_GRPC_URL = PUBLIC_URL`

## Risks / Trade-offs
- 风险：调用方仍使用旧 `mode` 参数。
  - Mitigation: 返回 400 并提供迁移提示（使用 local/remote 新入口）。
- 风险：local profile 网络名称与环境不一致。
  - Mitigation: 支持 `LUNAFOX_AGENT_DOCKER_NETWORK` 覆盖；默认 `lunafox_network`；缺失即失败。
- 风险：remote profile 下 PUBLIC_URL 配置错误。
  - Mitigation: 保持现有 PUBLIC_URL 校验与安装前提示。

## Migration Plan
1. 引入双入口路由并在 handler 中按 profile 渲染端点。
2. 删除 `mode` 分支逻辑，新增 `mode` 参数报错路径。
3. 更新安装脚本 local profile 的 network fail-fast 行为。
4. 更新前端 helper：安装命令改为调用 `remote` 入口（移除 query `mode`）。
5. 更新安装器 client：下载脚本改为调用 `local` 入口（移除 query `mode`）。
6. 更新测试和 UI 文案（本机安装 / 远程安装）。
7. 更新运维文档与 runbook，明确新入口和迁移方式。

## Open Questions
- 无
