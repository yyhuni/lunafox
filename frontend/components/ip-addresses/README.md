# IP Addresses Components

IP 地址列表属于标准 business-list 页面，表格外壳由 `IPAddressesDataTable` 和共享 `BusinessListDataTable` 拥有。

IP 地址列表使用 cursor-only pagination：当前已解析响应的 `nextPageToken` 决定下一页，
当前查询 token cache 中已访问的前页 token 决定上一页。搜索、端口筛选、排序、页大小或
父 scope 变化必须清空 cache 并回到第一页。结果总数仍显示为摘要，但不得推导总页数，
也不得显示页码、当前页或末页；首页按钮只清空 token 并重新请求第一页。

## Website Detail Scope

- Website detail passes an explicit read-only `WebsiteAssetScope`. Its target query starts with exact normalized `host=="<website.host>"`; ordinary `host="..."` contains search remains a separate user filter.
- The resulting IP rows are domain-resolution/observed-IP evidence for the Website host. This variant labels the IP column and search control as “域名解析 IP” / “Resolved IP Address”; it does not express IP ownership or exclusivity.
- This variant keeps search, port facet, pagination, row inspection, and copy, while omitting selection, export, delete, bulk actions, and their dialogs. Changing Website scope clears selection and resets page tokens.

## Loading Contract

- 正常页面初始 loading 必须复用 `IPAddressesDataTable`，通过 `loading` 和 `loadingRowCount` 交给共享表格渲染骨架行。
- `IPAddressesViewLoadingState` 必须接收完整 `IPAddressesViewState`，并把搜索、端口筛选、排序、分页、列定义传给同一个表格 owner。
- `IPAddressesViewRouteFallback` 仅供目标/扫描详情 layout 自己的完整 shell loading 使用，不参与 Sidebar 软导航。它必须用 `createIPAddressColumns` 和 `IPAddressesDataTable loading` 复现真实列、端口筛选与分页结构，不得退回通用 `DataTableSkeleton`。
- 解析态 route page 与路由 fallback 必须共同复用 `DetailAssetContentFrame` 的标准内容 gutter；普通 query loading state 不得再次嵌套该 frame。
- 目标详情的首屏 route fallback 在目标摘要可用时使用 `summary.ips` 推导 loading 行数，限制在 1 到默认页大小 10 行；摘要未到达或为空时保留 1 行结构。这个值只控制占位几何，不代表已解析的列表总数。
- IP 表格的 initial loading 必须通过共享 `initialLoadingToolbarFilterCount: 1` 保留端口筛选槽位；不要在 route skeleton 外部用额外高度补偿窄屏 toolbar 换行。
- 不要在 detail-layout loading state 里复制 toolbar 按钮数、列数、分页或行高。真实表格结构变化时，loading 态应自动跟随真实表格。
