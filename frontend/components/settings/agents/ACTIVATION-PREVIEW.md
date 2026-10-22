# Agent 扩容交互预览

正式 `/settings/agents/` 已使用 Server 的离线授权投影及激活接口。这里保留两种开发预览：独立交互原型，以及复用正式 Agent 页面组件的 MSW 网络工作台。真实签名、持久化和注册额度由 Server 负责；下列 `LF-DEMO-*` 只由开发 mock 识别，真实 Server 不接受。

## 启动与入口

```sh
cd frontend
pnpm dev:mock:noauth --port 3017
```

- 完整网络工作台：`http://localhost:3017/prototypes/agent-activation/workspace/`。
- 独立内存原型：`http://localhost:3017/prototypes/agent-activation/`。
- 旧 `/prototypes/agent-activation/toolbar/` 重定向至工作台，不保留工具栏额度方案。

这些 prototype 路由不进入正式导航，仅开发 mock 模式可用，其余环境返回 404。正式页沿用认证保护。

## 完整网络工作台

页面走 `components → hooks → services → api-client → MSW`，与正式页面使用相同的授权组件、配额保护和按需横幅。组件不判断 Key 算法，不区分 mock/real。MSW 拦截 `POST /v1/admin/agentLicenses/current:activate`，直接返回 `agentLicenses/current` 资源；summary 的 `license` 投影与它共享有效额度。旧 Server 未携带该投影时，前端不开放授权操作。

当前 fixture 在 2026-10-19 UTC 签发，示例额度均为 10：

| 示例 Key | 月数 | 固定到期时间（UTC） |
| --- | --- | --- |
| `LF-DEMO-1M` | 1 | 2026-11-19 00:00:00 |
| `LF-DEMO-2M` | 2 | 2026-12-19 00:00:00 |
| `LF-DEMO-3M` | 3 | 2027-01-19 00:00:00 |

- 相同 Key 重复导入、重新加载或换到另一个 mock 实例，都不重置到期时间。到期后必须换用尚有效的 Key；未来维护示例时需显式更新固定 fixture 日期。
- `LF-DEMO-EXPIRED` 返回过期错误；`LF-DEMO-NOT-YET-VALID` 返回尚未生效错误；其他输入返回无效 Key。失败保留输入及原授权。
- 授权状态只存在 mock 模块内存中，SPA 导航保留，完整刷新恢复对应场景的初始授权；刷新不会重新签发示例 Key。
- `happy`/`empty` 为基础授权，`stress` 预装 3 个月授权，`edge` 为已到期授权，`error` 在提交时返回 503。场景通过启动时的 `NEXT_PUBLIC_MOCK_SCENARIO` 选择。
- 工作台保留原有 10 节点 fixture，默认 10/3，激活后 10/10 仍满额。不要缩减数据只为让添加按钮可用。

横幅仅在满额、超额或到期时显示，未满且未到期时隐藏。工具栏额度/查看授权按钮、骨架占位和手动展开状态均已删除。横幅使用自然文字高度与共享间距；“如何获取 Key？”展开赞助、提交 Bug、参与引擎优化等贡献途径，并说明均需联系作者获取 Key；不展示未提供的联系方式。

提交中禁止重复提交与关闭；成功后 hook 等待 summary 刷新，额度以摘要为准。若新额度使横幅隐藏，成功弹窗继续保持挂载。刷新失败保留旧额度及现有 stale 反馈，不把激活响应直接覆盖为 summary。

## 独立内存原型

独立原型只用于演示 3/3 → 3/10 → 添加第 4 个节点，以及到期后保留 4/3 存量的视觉流程。弹窗可点击 1/2/3 个月示例，状态切换器可模拟到期，删除也只影响内存 fixture。关闭重开清空输入，刷新恢复 3/3。其固定预览日期与本地 state 不进入生产 service。

新增节点表单不得使用会遮蔽 DOM `nodeName` 属性的字段名，否则 Base UI 焦点管理与 React 事件会报错，提交可能退回浏览器原生 GET 导航。交互回归保护正常添加、额度更新和弹窗关闭。

## 验证范围

授权 service 测试保护固定 custom method、严格响应解析和空 Key fast-fail；network 测试保护 1/2/3 个月、同 Key 重放、失败保留、未生效和到期投影；flow 测试保护输入保留、pending、摘要刷新和横幅隐藏后的成功反馈。正式 AgentList 回归保护默认按需显示与旧 Server 兼容。

UI 继续复用共享 Dialog、Button、Badge、Textarea、Collapsible、typography、status helpers 和既有三个同步 section handoff，无新增 UI foundation 例外。纯文案和固定点数不单独新增机械测试。生产签名算法和注册并发由后端测试验证，MSW 不充当安全边界。

沿用的工具提示：Vite 对现有 `vitest.config.ts` CommonJS/ESM 组合提示未来兼容问题；MSW 2.15.0 对仓库已有 2.14.6 worker 提示版本不一致。owner 为 frontend tooling，状态为 inherited，恢复路径是在独立工具维护变更中迁移配置或按锁定版本重建 worker 并运行 network 回归；本次无相关依赖或 worker 修改。
