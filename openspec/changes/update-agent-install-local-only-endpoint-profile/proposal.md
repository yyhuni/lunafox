# Change: Agent 安装脚本改为双入口预设（零手填）并移除 mode 参数

## Why
当前安装链路存在三类问题：
- 用户心智负担高：若要求手动填写 `REGISTER_URL`、`RUNTIME_GRPC_URL`，操作复杂且易错。
- 行为不透明：`mode=local|remote` 属于隐式分支，排障时难以快速定位端点来源。
- 远程场景风险：远程 VPS agent 若固定走 `http://server`，跨机网络不可达会导致通信失败。

目标是在不增加用户输入的前提下，保持本地与远程部署都可用，并使端点行为显式、可预测、无回退。

## What Changes
- 提供两个明确安装入口（由 UI 按钮或 API 显式选择）：
  - `GET /api/agent/install-script/local?token=...`
  - `GET /api/agent/install-script/remote?token=...`
- 明确入口归属契约（避免跨场景误用）：
  - 项目安装器（`tools/installer`）只调用 `local` 入口。
  - 前端 Agent 安装弹窗只调用 `remote` 入口。
- 端点由服务端按入口预设渲染，用户无需手填变量：
  - `local` 入口：
    - `REGISTER_URL` -> `PUBLIC_URL`
    - `RUNTIME_GRPC_URL` -> `http://server:<SERVER_GRPC_PORT>`
  - `remote` 入口：
    - `REGISTER_URL` -> `PUBLIC_URL`
    - `RUNTIME_GRPC_URL` -> `PUBLIC_URL`
- 删除 `mode` 兼容语义：请求携带 `mode` 参数时返回 400，且不做任何回退。
- `local` 入口脚本强化前置校验：Docker network（默认 `lunafox_network`）缺失时失败退出，不再降级到默认 bridge。
- 更新测试、文档与运维说明，统一强调“无兼容回退”。

## Impact
- Affected specs: `runtime-communication-grpc`（delta）
- Affected code:
  - `server/internal/modules/agent/handler/agent_registration_handler.go`
  - `server/internal/modules/agent/handler/agent_handler.go`
  - `server/internal/modules/agent/router/routes.go`
  - `server/internal/modules/agent/install/templates/agent_install.sh`
  - `server/internal/modules/agent/handler/agent_registration_handler_test.go`
  - `frontend/lib/agent-install-helpers.ts`
  - `frontend/lib/__tests__/agent-install-helpers.test.ts`
  - `tools/installer/internal/agent/client.go`
  - `tools/installer/internal/agent/client_test.go`
  - `tools/installer/internal/steps/step_agent.go`
  - `docs/` 下安装与部署文档
- Deployment impact:
  - 本地部署通过 `local` 入口保持容器内网通信。
  - 远程 VPS 部署通过 `remote` 入口使用公网可达地址。
  - 历史 `mode` 调用会失败，需要调用方迁移到双入口。
