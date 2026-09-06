# Endpoints Components

URL 列表属于标准 business-list 页面，表格外壳由 `EndpointsDataTable` 和共享 `BusinessListDataTable` 拥有。

URL 列表使用 cursor-only pagination。下一页只可由当前已解析响应的 `nextPageToken`
启用，上一页只可由当前查询 token cache 中已访问的前页 token 启用。搜索、筛选、排序、
页大小或父 scope 变化必须清空 cache 并请求第一页；`totalSize` 只保留为结果摘要，不能推导
页数或恢复页码、当前页、末页导航；首页按钮只清空 token 并重新请求第一页。

## Filter Contract

- 目标 URL 与扫描历史 URL 页面共用 `EndpointsDataTable` 的一个聚合筛选面板；状态码、技术栈、Web 服务器、内容类型和虚拟主机不得重新渲染为独立 trigger。
- 面板选择为草稿：左侧分类计数即时反映草稿，只有点击“应用”才通过既有回调提交查询并刷新列表；“重置”只清空草稿。
- 已应用条件只通过聚合“筛选”按钮上的数量呈现；工具栏下方不得增加摘要行、chip、单项移除或面板外“重置”。

## Detail Inspection

- 主表默认隐藏可由 URL 直接读取的主机名，以及响应头和响应体；三列仍可通过共享“显示列”菜单按需启用。
- URL 内容区以“HTTP 状态码 + URL”呈现，状态码不再占用独立列；状态码筛选仍由聚合筛选面板提供。
- 非交互 URL 行区域打开 `EndpointDetailDrawer`；复选框、链接、按钮和带 `data-row-click-exempt` 的后代必须保留自身操作，不能打开抽屉。
- 抽屉复用共享 `DetailDrawer`，按概览、技术栈、响应头、响应体纵向呈现；响应区域只在本地代码区滚动。

## Website Detail Scope

- Website detail supplies a read-only `WebsiteAssetScope`; the target query begins with mandatory `websiteUrl=="<website.url>"` before the ordinary aggregate filter.
- The Server evaluates that value only as a read-time HTTP(S) origin and path-segment boundary: root includes same-origin descendants, while `/a` includes `/a`, `/a/`, and `/a/...` but not `/ab`. Query and fragment are ignored only here; full raw URLs remain the persisted identity and ordinary URL filters use exact equality.
- This variant preserves search, facets, pagination, row drawer, and copy. It omits selection, export, add/delete, bulk actions, and dialogs; Target and Scan variants remain unchanged.

## Loading Contract

- 正常页面初始 loading 必须复用 `EndpointsDataTable`，通过 `loading` 和 `loadingRowCount` 交给共享表格渲染骨架行。
- `EndpointsDetailViewLoadingState` 必须接收完整 `EndpointsDetailViewState`，并把搜索、筛选、排序、分页、列定义传给同一个表格 owner。
- `EndpointsDetailViewRouteFallback` 仅供目标/扫描详情 layout 自己的完整 shell loading 使用，不参与 Sidebar 软导航。它必须用 `createEndpointColumns` 和 `EndpointsDataTable loading` 复现真实列、聚合筛选、列可见性与分页结构，不得退回通用 `DataTableSkeleton`。
- 解析态 route page 与路由 fallback 必须共同复用 `DetailAssetContentFrame` 的标准内容 gutter，普通 query loading state 不得再次嵌套该 frame；fallback 同时复用 `DEFAULT_ENDPOINT_COLUMN_VISIBILITY`，强制刷新时骨架与真实表格不得因容器或默认可见列不同而横向跳动。
- 不要在 detail-layout loading state 里复制 toolbar 按钮数、列数、分页或行高。真实表格结构变化时，loading 态应自动跟随真实表格。
