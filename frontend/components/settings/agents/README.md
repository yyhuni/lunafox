# Agent UI Rules

`frontend/components/settings/agents/` 负责 `/settings/agents` 页面及其局部交互。这里的页面运行在带固定侧栏的主壳层里，所以响应式判断应优先基于主内容容器，而不是整个视口。

## Product Terminology

- 设置导航、列表、详情和操作面向 Agent 节点；不要恢复独立的 Worker 产品实体或兼容字段。
- 架构视图使用 `Server -> Agent -> Engine Container`：Agent 管理本机 Engine execution，Engine 不是可独立管理的节点资源。
- 系统架构弹窗将单行关系说明作为画布底部悬浮信息展示；不要恢复底部“运行流程”时间线或使节点脱离其连接关系单独移动。
- 架构弹窗右侧内容区本身就是画布：不添加内边距或固定高度留白，拓扑按可用宽高等比缩放并居中。
- 架构弹窗以摘要栏 `264px` 加拓扑画布 `920px` 的自然宽度为上限，并将高度收敛到 `720px`；窄屏由视口上限继续约束。
- 架构拓扑是静态示意，Server、Agent 和 Engine 节点不得展示“在线”或“运行中”等实时运行状态。
- 架构图中的 Agent 节点由内容决定高度；连线锚点使用其紧凑几何框的底部内边距，避免卡片尺寸与回传连线脱节。
- Agent 卡片中，`runningTasks` 表示正在运行的任务数，`taskSlotsUsed` 表示已占用的执行槽位；两者必须分别展示，不能再合并为一个 `running/maxTasks` 数值。顶部执行容量摘要只呈现槽位维度，不得把已占用槽位改名为运行中任务。
- 卡片的 CPU、内存、磁盘、任务数、槽位和 uptime 只来自未过期的 Redis heartbeat projection。heartbeat 缺失或读取失败时必须明确显示“实时指标不可用”，不渲染任何旧指标、`-` 占位或 `maxTasks` 组合值；独立持久化的 `lastHeartbeat` 仍须继续展示为最后观测时间，且不得用 PostgreSQL runtime snapshot 回填 heartbeat。

## Query Contract

- `/settings/agents/` 是已迁移的 controlled card-list variant，不是表格页，但列表数据必须走统一后端查询：`pageSize`、`pageToken`、`filter`、`orderBy`。
- 搜索框只做普通 contains 搜索，编译到 `displayName`、`observedHostname`、`connectionIp` 的 OR filter；不要暴露 SmartFilter / FOFA 语法，也不要把 `instanceId`、版本、CPU、内存、磁盘或日志内容加入默认搜索。
- `connectionIp` 是所有 Agent 管理入口唯一的当前控制连接地址：有值时直接展示，离线空值显示“未连接”，在线空值显示“未观测到”。`observedSourceIp` 仅为 GeoIP/位置溯源，不能作为第二个管理地址或空值回退。
- 状态筛选 UI 保留“健康、预警、离线、未知”四桶，但请求必须显式编译为 `status` 与 `healthState` 的后端字段组合；不要发送 `statusBucket` / `agentStatusBucket` 这类前端临时字段。
- `frontend/lib/agent-status-distribution.ts` 是 Agent collection 卡片与筛选四桶的语义 owner，不能各自把未知健康状态降级为健康或把离线与预警重复计数。全局汇总桶由 Server 的 `agentClusterSummaries/current` 直接提供，前端不得从当前 collection 页重新分类或覆盖结论。
- 筛选菜单选项来自 `useAgentFilterOptions("status")` 与 `useAgentFilterOptions("healthState")`，不能从当前页卡片推导完整 options。顶部 overview 必须使用共享的 `useAgentClusterSummary` / `agentClusterSummaries/current` 全集投影，不随列表搜索、筛选、翻页变化，也不得遍历 `nextPageToken` 或从当前页回退聚合。节点桶、通用执行容量、集群结论、原因码和位置覆盖均直接使用 Server 结果；首次失败显示可重试错误，后台刷新失败保留并明确标记最后成功快照，后续成功原子替换并清除过期提示。该快照只用于运维展示，不能进入选择、调度或任务领取控制。
- 排序只开放 `createdAt`，默认 `createdAt desc`；名称、主机名、IP、实例 ID、心跳、状态、健康状态和资源用量都不要显示排序 affordance。
- 列表请求不得发送旧参数 `page`、独立 `status`、`include=heartbeat`、`sort*` 或 `keyword`。

## Responsive Rules

- 优先使用 `@container/main` 相关容器查询类来切换 overview、toolbar 和卡片网格布局。
- 不要把这类页面写成只依赖 `xl:` / `2xl:` 的视口断点页面；侧栏会先吃掉一部分宽度。
- overview 标题行在主内容区达到 `@4xl/main` 时保持标题在左、刷新操作靠右；低于该阈值才上下堆叠，避免在常规桌面宽度浪费右侧操作位。
- overview 区由标题/刷新行和一条连续的四段集群摘要组成：依次表达 Agent 节点拥有量、Server 返回的集群状态、健康节点比例和执行容量；宽容器使用共享可见边界按四等分分隔，并将每段内容居中放入自己的区域，较窄容器按两列或单列换行。节点总数必须明确绑定到“Agent 节点”，容量摘要必须同时表达可用、已占用、不可用、超额占用和配置总槽位，避免孤立数字或状态图标语义冲突。
- toolbar 优先保证换行和缩放稳定，再考虑单行铺开。

## Guard Rails

- Agent 日志使用 `StructuredLogViewer` + `LiveLogSurface` 的 structured-live 组合。Agent 模块继续拥有日志 hook、筛选、来源/连接状态、错误映射和自动刷新，但不得本地复制 terminal body、viewport、焦点、跳到最新按钮或 footer shell；复制当前日志入口通过 `LiveLogSurface` 的 `topRightAction` 放在正文右上角，内容必须与解析后的可读日志行一致。
- Agent 日志抽屉的行数选择只允许 `100/200/500/1000/2000/5000`，由共享 `TerminalLogToolbar` 的 Base UI Select 呈现；默认 100，且只在当前抽屉会话有效。N 是浏览器保留的最新行窗口，不是后端单页大小或完整历史数。上滚只暂停自动滚屏，不能停掉自动刷新、follow 或轮询；关闭抽屉和切换 Agent 必须恢复 100。
- 安装命令的复制操作必须复用共享 `CopyButton`，并置于命令显示框的右上角；命令标题与文本需为该操作预留右侧空间。
- 安装器的网络、镜像与执行前提只在弹窗顶部 `DialogDescription` 说明一次；命令分区仅保留标题和命令输出，不要重复同一段环境说明。
- 安装器内容按连续分区组织：令牌、命令和步骤说明不各自再包独立卡片，而是由共享 `Separator` 分隔；仅命令输出框和状态告警保留边界，以维持操作与风险语义。
- 有效令牌的到期时间和“有效期内可重复使用”必须与状态 Badge 位于同一 metadata 行，并均通过圆点与前项分隔；窄屏可换行，但不要退回为独立说明行。
- 安装器连接反馈只在弹窗打开且观察未结束时，每约 2 秒读取当前非密钥资源 `agentRegistrationTokens/{registrationToken}`，并以其完整 Agent 投影替换当前结果；不得读取 Agent collection、建立时间窗口 baseline 或用 ID 差分猜测归因。令牌有效期内即使已知 Agent 全部在线也继续观察；令牌过期后，在非空且全部在线的成功投影或两分钟实现级宽限期限二者先到时停止。查询失败保留上次成功投影并按轮询间隔恢复；关闭、换令牌和卸载必须取消请求。最终结果由页面状态保留，同页关闭再打开时明确显示为已过期、非实时且不得恢复请求；只有生成新令牌或页面生命周期结束才重置。
- Agent 日志行的可见格式和复制顺序由 `StructuredLogViewer` 统一拥有；不要在抽屉内恢复 Badge、固定宽度级别列或本地字段重排。
- Agent 日志的 latest-N 仍完整保存在内存中供筛选、计数和生成当前可见范围的格式化复制文本；超过共享阈值后，`StructuredLogViewer` 必须复用 `LiveLogSurface` 的同一个 `viewportRef`，只挂载可见行和有限 overscan，并按换行后的真实行高测量。不要增加抽屉局部滚动容器。虚拟化后原生拖选不保证跨未挂载行，当前筛选结果复制继续走正文右上角的复制当前日志按钮。
- Agent follow 轮询没有新增唯一日志时，正文 `lines` 必须保持原数组引用；连接、gap 和 caught-up metadata 可以更新，但空轮询不得重新提交正文或触发自动滚屏布局。
- 日志抽屉的 `initialFocus` 指向共享 viewport，并通过 `LiveLogSurface` 的 `focusable` 模式获得统一焦点反馈；不要在 Agent 模块恢复本地 `tabIndex` 或 focus ring class。

- 不要恢复到需要多个固定轨道同时满足的复杂大 grid，例如：
  - `xl:grid-cols-[repeat(5,minmax(0,1fr))_minmax(300px,1.6fr)_180px]`
  - `@7xl/main:grid-cols-[minmax(0,1.9fr)_minmax(280px,1fr)_180px]`
- 宽屏 overview 使用方案 C 的一行连续摘要：标题/刷新保持独立行，摘要依次放置节点拥有量、Server 集群结论、健康节点和执行容量；不要恢复统计卡、容量卡或快捷入口卡组成的多轨 grid。Agent 拥有量图标保持中性，不能在集群为“需关注”时显示绿色成功标记。
- 搜索区、筛选区、卡片标题区在 flex/grid 容器内要优先补齐 `min-w-0`。
- 正式页面和 loading state 要使用同一套断点策略，避免加载前后布局跳变过大。
- overview 状态桶的 loading 文本必须保留与真实态一致的“本地化标签 + 数值”宽度；只占位标签会在 `390px` 窄屏少换一行，使 skeleton 与真实 summary 相差一行高度。
- 顶部 overview 的真实态和 loading 态必须由 `AgentOverviewSection` 同一个 owner 派生；`AgentOverviewLoadingState` 只能包装 `<AgentOverviewSection loading />`，不要在 `agent-list.tsx` 外维护第二套状态摘要或执行容量结构。
- overview 标题行、状态/容量摘要和 toolbar 外壳必须从 `agent-layout-contract.ts` 共享；不要在 `agent-list.tsx` 和独立 page skeleton 文件里各自手写同一组 flex/grid class。
- overview 刷新/更新时间、筛选、toolbar 操作和卡片操作菜单等 action-like 占位必须走 `ActionSkeleton` 对齐真实 `Button` / action menu 尺寸契约；不要在 loading state 中回退到本地 `Skeleton h-* w-* rounded-*` 方块。真实更新时间从客户端首次挂载时间开始，手动刷新后更新为集群 summary、当前参数列表和筛选选项请求均结束的时间；不得使用最新 Agent 心跳或自动 polling 时间。
- overview 的手动刷新入口必须使用共享 `PageRefreshStatusButton`。按钮时间在客户端首次挂载后表示页面时间，手动刷新后表示 summary、当前参数列表和 `status`/`healthState` 筛选选项查询均结束的刷新完成时间，格式固定为本地时区 `YYYY/MM/DD HH:mm:ss`；该操作不重置当前搜索、筛选或分页。不得用 summary 的 `generatedAt`、最新 Agent 心跳或自动 polling fetching 状态代替页面刷新状态。Agent 心跳新鲜度继续由 Agent 卡片所有。
- Agent 卡片 loading visual 必须由 `AgentCardCompactLoadingState` 承载，`AgentCardCompact` 的 loading variant 只委托给该轻量 visual；`AgentCardsLoadingState` 只负责渲染结果 grid、固定数量的 Agent loading cards，以及一个与末尾新增 Agent 操作槽配对的共享 `ActionSkeleton` 几何槽。该操作槽占位不计作 Agent。卡片 shell、header、body、footer、信息区和 metrics 行几何都要从真实卡片 owner 与 `agent-layout-contract.ts` 共享，不要重新手写一套 compact card class；资源指标行必须使用 `SegmentedMetricProgress` 的原生 `loading` 变体，不能再以独立固定高度方块近似。标题/IP 的 loading 容器必须保留正式 typography class 的原始顺序，不能经 `cn`/`twMerge` 合并，否则 `text-sm` 会移除 `leading-none` 并改变卡片高度。
- 紧凑 Agent 卡片的最后心跳绝对时间必须保留秒；同一年可隐藏年份，跨年保留年份。运行中/槽位与心跳/已运行两行必须复用同一套两列网格，保持列对齐；心跳时间只可在左列空间不足时自身截断，并保留完整 `title`，不得为时间显示而挤压或移动右列的已运行槽位。
- `agent-overview-loading-state.tsx` 与 `agent-card-compact-loading-state.tsx` 分别持有 overview 和 compact card 的纯 loading visual；`AgentList` 内的 resolved section owner 继续复用这些分支。Sidebar 或 protected shell 不得在这些页面自有状态之前插入另一套 Agent 骨架。
- 不要 reintroduce `agent-page-skeleton.tsx`、`AgentPageSkeleton` 或整页替换式 agent skeleton；`/settings/agents` 的初始 loading 由 `AgentList` 内的 `overview`、`toolbar`、`results` 三个同步 section owner 承担。
- 这三个 `ContentHandoff` 必须显式保持 `layer="section"`，并作为同一个
  `/settings/agents` 首屏契约中的三个 paired `surface` owner 登记。它们还必须在
  真实区块根节点上分别配对 `agent-list-overview-region`、
  `agent-list-toolbar-region`、`agent-list-results-primary-region`；不要把它们
  提升为一个 workspace/page skeleton，也不要把分页另做成只在有数据时才存在的槽位。
- `/settings/agents` 是 sectioned dashboard，不是 `title -> toolbar -> results`
  的同层 staged reveal 样板。`overview`、`toolbar`、`results` 可以保持分区 owner，
  但它们必须共享同一个初始 ready gate；如果将来浏览器时间线证据显示三块在首屏依次
  亮相，再考虑收敛为更高层 owner，而不是先为了形式统一合并成 page-wide skeleton。
- 首次 section handoff 以几何稳定为准，不以 query 仅仅成功返回为准。`overview` 和 `results` owner 必须继续走 shared `ContentHandoff`，并在真实内容可挂载后通过 `mountContentWhileLoading` 先隐藏挂载，再等待双 `requestAnimationFrame` 后才允许 skeleton 退出。
- 状态筛选使用共享 `DataTableFacetedFilter`，保持与目标列表一致的虚线筛选按钮、多选弹层和选中数量 affordance；不要在页面内恢复本地 `Select` 状态下拉。
- 紧凑 Agent 卡片右上角的三点操作菜单默认以触发器左缘对齐、向右下展开（`DenseRowActionMenu align="start"`）；共享 Positioner 在右侧空间不足时翻转至左侧，不要在卡片内手写视口或坐标判断。
- 状态筛选继续使用健康、预警、离线、未知四类；不要退回只筛在线/离线的 runtime 状态。执行容量不改变该筛选语义。
- 执行容量完全使用 Server summary：只有 `online + healthy` 且 15 秒执行观测仍新鲜的 Agent 才贡献占用和可用配置槽位，其余配置槽位计入不可用；超出配置上限的使用量单独显示为 `overcommittedSlots`。容量条颜色必须使用共享 `getStatusToneBgClass`，下方文字显示可用、已占用、不可用和配置总槽位，灰色不可用段仍保留在条内。第三段使用 `healthyCount / totalNodes`，不从 collection 推导在线数。集群状态与有序原因码直接本地化 Server 返回值；任何正的低可用率都不得由前端自行降级。
- 当前结果页存在 Agent 节点时，结果网格必须在最后一个真实卡片之后展示一个与真实卡片同宽同高的 `AgentExpansionSlot`，占用一个普通网格单元并复用现有安装器入口。该槽位是创建操作，不是 Agent 数据或虚构节点；初始加载只允许用一个无业务内容的共享 `ActionSkeleton` 保留其 grid 几何，空集群和无匹配结果继续使用各自的空状态入口。
- 不要在 `AgentList` 里手写 `bg-success` / `bg-warning` / `bg-error` / `bg-muted-*` 分布颜色映射；新增状态先扩展 `AgentStatusDistributionStatus` 和对应 helper。
- 不要为了让测试看到 helper 而把 Tailwind class 字符串渲染到 DOM，包括 `sr-only`。可访问文本只能是用户可理解的状态与数量。

## Audited Evidence

- desktop `1280x720` timeline:
  - `700ms`: only `initial-boot`
  - `1300ms`: `agent-list-overview` / `agent-list-toolbar` / `agent-list-results` are in the same initial section phase (`loading` or `handoff`, depending on sampled readiness timing)
  - `2200ms`: the same three owners all in `content`
- mobile `390x844` timeline:
  - `700ms`: only `initial-boot`
  - `1300ms`: `agent-list-overview` / `agent-list-toolbar` / `agent-list-results` are in the same initial section phase (`loading` or `handoff`, depending on sampled readiness timing)
  - `2200ms`: the same three owners all in `content`

当前结论：这是同步 section handoff，不是 `overview -> toolbar -> results`
的同层分段亮相。

## Verification

- 代码守卫：更新 `__tests__/agent-list.contract.test.ts` 与 `__tests__/agent-status.contract.test.ts`
- 基础回归：`cd frontend && pnpm run test`
- 浏览器验收：检查 `/settings/agents/` 在窄桌面宽度和常规桌面宽度下的 overview、toolbar 和 results handoff 高度；若怀疑同层分段亮相，用 `LOADING_SMOKE_TARGET_IDS=route_settings_agents LOADING_SMOKE_TIMELINE_MS=700,1300,2200,4000 node scripts/run-loading-handoff-smoke.mjs` 抓共享时间线
