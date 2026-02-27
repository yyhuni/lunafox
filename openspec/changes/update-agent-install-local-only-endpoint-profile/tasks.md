## 1. Spec & behavior contract
- [ ] 1.1 确认双入口契约：`/api/agent/install-script/local` 与 `/api/agent/install-script/remote`
- [ ] 1.2 明确端点矩阵（local 与 remote 预设），并保证用户零手填运行期端点变量
- [ ] 1.3 实现 `mode` 删除语义：请求携带 `mode` 参数时返回 400（无回退）

## 2. Server rendering changes
- [ ] 2.1 新增并接入 local/remote 安装脚本路由
- [ ] 2.2 从 `mode` 分支切换为“按入口 profile 渲染”逻辑
- [ ] 2.3 保持 `PUBLIC_URL` 校验，缺失或非法时明确报错

## 3. Install script changes
- [ ] 3.1 local profile 下要求 Docker network 存在（默认 `lunafox_network`），缺失即失败退出
- [ ] 3.2 删除 network 缺失时降级 default bridge 的逻辑
- [ ] 3.3 保持 remote profile 不依赖 `server` 内网主机名

## 4. Caller ownership changes
- [ ] 4.1 更新前端安装命令 helper：固定调用 `/api/agent/install-script/remote`
- [ ] 4.2 更新安装器下载脚本 client：固定调用 `/api/agent/install-script/local`
- [ ] 4.3 删除前端与安装器中 query `mode` 拼接逻辑

## 5. Tests
- [ ] 5.1 更新 `agent_registration_handler_test.go`：local/remote 双入口端点渲染断言
- [ ] 5.2 新增 mode 参数报错测试：传入 mode 时返回 400
- [ ] 5.3 新增安装脚本片段断言：local profile 网络缺失 fail-fast
- [ ] 5.4 更新 `frontend/lib/__tests__/agent-install-helpers.test.ts`：断言命中 remote 入口且无 mode 参数
- [ ] 5.5 更新 `tools/installer/internal/agent/client_test.go`：断言命中 local 入口且无 mode 参数
- [ ] 5.6 运行 `go test ./internal/modules/agent/handler ./internal/bootstrap` 验证回归

## 6. Docs & validation
- [ ] 6.1 更新安装/运维文档，改为“本机安装 / 远程安装”双入口说明
- [ ] 6.2 更新迁移说明：旧 `mode` 参数已弃用并会失败
- [ ] 6.3 明确入口归属：安装器=local、前端=remote
- [ ] 6.4 运行 `openspec validate update-agent-install-local-only-endpoint-profile --strict --no-interactive`
