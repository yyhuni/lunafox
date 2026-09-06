# Directories Components

目录列表属于标准 business-list 页面，表格外壳由 `DirectoriesDataTable` 和共享 `BusinessListDataTable` 拥有。

目录列表使用 cursor-only pagination：当前已解析响应的 `nextPageToken` 控制下一页，
当前查询 token cache 中已访问的前页 token 控制上一页。搜索、筛选、排序、页大小或
父 scope 变化必须清空 cache 并回到第一页。`totalSize` 仅是结果摘要；不得据此推导页数，
也不得显示页码、当前页或末页；首页按钮只清空 token 并重新请求第一页。

状态与内容类型属于同一组结构化筛选，必须通过共享 `DataTableFacetPanel` 收束为一个“筛选”入口；不要在目录工具栏并列渲染多个独立分面筛选器。

## Read Contract

- `contentLength` 与纳秒 `duration` 的生产类型固定为 `string | null`。Service 只接受 canonical 非负 int64 十进制字符串或 `null`，不得经 JavaScript `number` 转换；仅在确需算术时使用 `BigInt`。
- 表格和 CSV 直接保留 Server 字符串。CSV 的 `duration` 仍是纳秒原值，不做毫秒换算；`"0"` 与 `null` 的含义不同。
- Directory 身份只使用原始 `url`；生产模型不包含 `words`、`lines` 或 `websiteUrl`。列表和导出不得因 originating Task/Scan 失败或取消而隐藏已确认结果。

## Website Detail Scope

- Website detail uses the same `DirectoriesView` with a read-only `WebsiteAssetScope`. Its target request starts with `websiteUrl=="<website.url>"`; this is a server-side read-time virtual Scope, not a persisted Directory field or a raw string prefix. It derives origin/effective-port/path hierarchy only for relation grouping, never rewrites the raw URL or changes ordinary exact URL filtering.
- Scope changes clear selection and page tokens. Search and facets only narrow the Website range; `totalSize`, ordering, and pagination remain scoped.
- Website context retains filtering, pagination, and row inspection, but does not render selection, export, add/delete, bulk actions, or confirmation-dialog owners. Target and Scan contexts keep their management behavior.

## Loading Contract

- 正常页面初始 loading 必须复用 `DirectoriesDataTable`，通过 `loading` 和 `loadingRowCount` 交给共享表格渲染骨架行。
- `DirectoriesViewLoadingState` 必须接收完整 `DirectoriesViewState`，并把搜索、筛选、排序、分页、列定义传给同一个表格 owner。
- `DirectoriesViewRouteFallback` 仅供目标/扫描详情 layout 自己的完整 shell loading 使用，不参与 Sidebar 软导航；它必须复用 `DirectoriesDataTable` loading mode 与 `useDirectoryTableColumns`。
- 不要在 detail-layout loading state 里复制 toolbar 按钮数、列数、分页或行高。真实表格结构变化时，loading 态应自动跟随真实表格。
