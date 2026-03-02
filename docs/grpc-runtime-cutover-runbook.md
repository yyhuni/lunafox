# Runtime gRPC Big-Bang 切换 Runbook

本文档用于落实 OpenSpec `refactor-runtime-communication-to-grpc` 的发布策略任务 `2.1 ~ 2.4`。

## 1. 切换窗口、版本基线与回滚（2.1）

### 1.1 一次性切换窗口

- 切换方式：`big-bang`，不保留运行时双栈。
- 推荐窗口：低峰时段，预留至少 `90` 分钟可回退窗口。
- 参与角色：发布负责人、后端值班、平台值班、观测值班。

### 1.2 版本基线（必须严格一致）

- `server`、`agent`、`worker` 必须使用同一 `RELEASE_VERSION` 发布。
- 不允许混部：
- 禁止新 `server` + 旧 `agent/worker`。
- 禁止旧 `server` + 新 `agent/worker`。
- gRPC 端口基线：
- `server` 内部监听 `SERVER_GRPC_PORT`（默认 `9090`）。
- 对外统一入口仍为 `443`（Nginx/LB 转发）。

### 1.3 回滚预案（整版回退）

- 回滚触发条件（任一满足即回滚）：
- Agent 无法稳定建立 runtime stream（连续重连 > 10 分钟）。
- 任务分发、取消、结果回传任一关键链路不可用。
- 管理面 HTTP API 出现阻断级故障。
- 回滚动作：
- 将 `server`、`agent`、`worker` 同时回退到上一个稳定 `RELEASE_VERSION`。
- 恢复旧版 Nginx/LB 配置（同一变更单内保留上一版本配置快照）。
- 回滚完成后执行最小验证：
- `GET /health` 返回 `200`。
- 管理面登录与扫描列表可用。

## 2. 预发布全链路演练（2.2）

在预发布环境执行一次完整 rehearsal，并产出勾选记录（建议附到变更单）。

### 2.1 演练检查项

- [ ] 所有组件升级到同一 `RELEASE_VERSION`。
- [ ] Agent runtime gRPC stream 连通并稳定（无异常重连风暴）。
- [ ] 任务分发成功（agent 收到 task_assign）。
- [ ] 任务取消成功（server 下发 task_cancel，agent/worker 正确收敛）。
- [ ] 结果回传成功（worker -> agent UDS gRPC -> server 数据代理落库）。
- [ ] 管理面关键 API 无回归（登录、目标、扫描、漏洞列表）。

### 2.2 最小验证命令

```bash
# 1) 三端测试
cd server && go test ./...
cd ../agent && go test ./...
cd ../worker && go test ./...

# 2) OpenSpec 变更校验
cd .. && openspec validate refactor-runtime-communication-to-grpc --strict --no-interactive
```

## 3. 发布冻结与门禁（2.3）

### 3.1 冻结策略

- 切换前至少 `24` 小时冻结以下范围：
- `server/internal/grpc/runtime/**`
- `agent/internal/runtime/**`
- `worker/internal/server/runtime_client.go`
- `docker/nginx/nginx.conf`
- 仅允许修复阻断级问题，且必须二次评审。

### 3.2 放行门禁

- 必须同时满足：
- 预发布 rehearsal 全部通过。
- `server/agent/worker` 的 `go test ./...` 全绿。
- OpenSpec 严格校验通过。
- Nginx/LB 443 入口 HTTP + gRPC 双通道验证通过。

## 4. 443 统一入口的 Nginx/LB 转发（2.4）

### 4.1 Nginx 配置要求

仓库内已落地配置（`docker/nginx/nginx.conf`）：

- HTTP upstream：`server:8080`
- gRPC upstream：`server:9090`
- gRPC 路由前缀：`/lunafox.runtime.v1.(AgentRuntimeService|AgentDataProxyService)/`

### 4.2 验证步骤

```bash
# 1) 校验 compose 配置可解析
docker compose -f docker/docker-compose.yml config >/dev/null

# 2) HTTP 管理面健康检查
curl -sk https://<PUBLIC_HOST>:443/health

# 3) gRPC 入口连通性（未授权应返回 Unauthenticated）
grpcurl -insecure \
  -H "x-worker-token: invalid-token" \
  -d '{"scan_id":1,"tool_name":"subfinder"}' \
  <PUBLIC_HOST>:443 \
  lunafox.runtime.v1.AgentDataProxyService/GetProviderConfig
```

预期：

- 第 2 步返回 `200`。
- 第 3 步返回 gRPC `Unauthenticated`（证明 443 -> gRPC upstream 路由生效，且鉴权链路生效）。

## 5. CI 自动生成 Release Manifest（推荐）

在 CI 内统一从构建产物注入三个环境变量，再自动生成 `release.manifest.yaml`：

- `RELEASE_VERSION`
- `AGENT_IMAGE_REF`（必须 digest）
- `WORKER_IMAGE_REF`（必须 digest）

示例（GitHub Actions）：

```yaml
- name: Generate release manifest
  env:
    RELEASE_VERSION: ${{ github.ref_name }}
    AGENT_IMAGE_REF: ${{ steps.build_agent.outputs.image_ref }}
    WORKER_IMAGE_REF: ${{ steps.build_worker.outputs.image_ref }}
  run: |
    make gen-release-manifest
    make verify-release-contract
```

说明：

- `make gen-release-manifest` 会自动把 `agentVersion/workerVersion` 设为 `RELEASE_VERSION`。
- 生成后会调用 `scripts/ci/verify-release-contract.sh` 做发布契约校验。
- 发布流水线会自动产出并发布 `release.manifest.yaml`（GitHub/Gitee Release 附件），并同步到 `release-channel` 分支的 `manifests/` 目录（按 tag 文件 + stable/canary 别名文件）。
