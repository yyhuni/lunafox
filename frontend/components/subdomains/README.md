# Subdomains Components

子域名列表属于标准 business-list 页面，表格外壳由 `SubdomainsDataTable` 和共享 `BusinessListDataTable` 拥有。

## Cursor Pagination

目标和扫描子域名列表使用 cursor-only pagination。当前已解析响应的
`nextPageToken` 是下一页的唯一依据，当前查询 token cache 中已访问的前页 token
是上一页的唯一依据；搜索、排序、页大小或父 scope 变化必须清空 cache 并回到第一页。
保留结果总数和页大小选择，但不得由 `totalSize` 推导总页数，也不得显示页码、当前页或末页；首页按钮只清空 token 并重新请求第一页。

## Loading Contract

- 正常页面初始 loading 必须复用 `SubdomainsDataTable`，通过 `loading` 和 `loadingRowCount` 交给共享表格渲染骨架行。
- `SubdomainsDetailViewLoadingState` 必须接收完整 `SubdomainsDetailViewState`，并把搜索、排序、分页、列定义传给同一个表格 owner。
- `SubdomainsDetailViewRouteFallback` 仅供目标/扫描详情 layout 自己的完整 shell loading 使用，不参与 Sidebar 软导航。它必须用 `createSubdomainColumns` 和 `SubdomainsDataTable loading` 复现真实列、工具栏与分页结构，不得退回通用 `DataTableSkeleton`。
- 解析态 route page 与路由 fallback 必须共同复用 `DetailAssetContentFrame` 的标准内容 gutter，确保强制刷新期间的骨架与真实表格对齐；普通 query loading state 不得再次嵌套该 frame。
- 不要在 detail-layout loading state 里复制 toolbar 按钮数、列数、分页或行高。真实表格结构变化时，loading 态应自动跟随真实表格。
