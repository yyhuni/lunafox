# MCP 连接页面

`/settings/mcp/` 是侧边栏 System 分组中的直接目的地，复用应用现有壳层。顶栏不再提供 MCP 按钮或配置 Popover。

- `mcp-connection-settings.tsx` 使用原有 `useMcpKeyStatus` / `useGenerateMcpKey`。地址始终是当前 origin 的 `/mcp`；服务端签发的一次性密钥只放在页面 state，离页或主动隐藏后不可重新读取。Query cache 只持有 configured/时间等非秘密元数据。禁止将密钥写入 URL、持久化存储或连接图。
- `mcp-client-configuration.ts` 生成四客户端的现有原生配置，认证只进入 Authorization header。轮换继续使用共享 ConfirmDialog，取消不调用接口。
- 页面聚焦真实配置，不提供模拟工具调用、示例结果或调用演示的播放控制。`mcp-connection-flow.tsx` 只表达客户端、MCP 与三类安全能力之间的关系，随当前客户端高亮输入路径，不代表真实连通性状态，不访问服务或密钥。
- 桌面连接图默认自动流动三个错开的 SVG 粒子，路径随客户端更新。浏览器原生 SVG 时间线驱动位移，不逐帧更新 React。实时媒体偏好、IntersectionObserver 和页面可见性控制粒子挂载，reduced motion、窄屏、离开视口或隐藏页面时卸载；卸载时清理订阅。
- 配置区用统一 Tabs 选择客户端；宽屏地址/密钥与代码分栏，窄屏按选择、地址/密钥、代码的顺序堆叠。切换客户端时密钥 state 保持在配置面板上，避免 Tab panel 切换导致一次性秘密丢失。代码自身滚动，不撑开页面。
- SVG 的 viewBox、节点坐标和连线路径属于本组件的图示几何；文字、颜色、外壳和按钮继续由共享 typography、theme tokens、Card、Badge、Button 拥有。连续 3.6 秒线性路径运动是本图的可视化例外，统一 ledger 以 `motion-mcp-connection-flow` 记录精确范围、owner、复核条件和恢复路径。
- 本页没有新增后端接口。真实模式使用 `pnpm dev --port 4319`，保留正常登录认证；Next 将 `/v1/*` 和精确 `/mcp` 转发到 `http://${API_HOST:-localhost}:8080`。密钥状态读取 `/v1/users/me/mcpKey`，生成/轮换调用 `/v1/users/me:generateMcpKey`。真实 Vercel 部署的两个路径使用相同的服务端 `BACKEND_URL`；Docker HTTPS 入口沿用 Nginx 的现有代理。
- `pnpm dev:mock:noauth --port 4319` 仅用于视觉预览，密钥请求由现有网络层 mock 拦截，Next 不代理到真实服务；这不构成真实后端验收。切换模式需重启 Next。真实联调应使用隔离本地账号验证状态、生成和 MCP 握手/工具清单，不为验收轮换已有业务账号的密钥或启动扫描。
- 页面配置壳层和连接图同步渲染，沿用 protected shell 的 `auth-layout-route-content` 首屏 owner；密钥状态只在 Input 内显示等待，不增加页面 skeleton。对应路由登记在现有 `scripts/loading-route-contracts.mjs`。

验证：`pnpm exec vitest run components/mcp/__tests__ components/__tests__/unified-header.contract.test.ts`、`pnpm typecheck`、`pnpm check:ui-foundation`。测试保护密钥生命周期、轮换确认、错误重试、客户端原生格式和切换客户端保留密钥；纯视觉布局不新增断言。浏览器检查真实侧边栏跳转、配置复制、主题和桌面/窄屏布局。
