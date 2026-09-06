# Vulnerabilities Components

漏洞列表属于标准 business-list 页面，表格外壳由 `VulnerabilitiesDataTable` 和共享 `BusinessListDataTable` 拥有。

漏洞详情只通过整行激活，漏洞类型单元格保持只读文本，不再提供重复的行内详情按钮。

Target、scan、global 和 website-scoped 漏洞列表只使用相邻 cursor 导航：总数是信息摘要，
当前响应 token 授权下一页，当前查询 token cache 授权上一页。纵向 header 不得把总数转换成页码、
首页 reset 或相邻 token 导航；不得提供末页、页码或当前页导航。

## Website Detail Scope

- Website detail passes a read-only `WebsiteAssetScope` to the Target vulnerability workspace. It prefixes the collection filter with `websiteUrl=="<website.url>"`, which the Server treats as a read-time same-origin and path-segment boundary without adding a Website foreign key or URL split fields. Query and fragment are ignored only for that relation view; ordinary Vulnerability URL identity and filtering retain exact raw-string equality.
- The scoped variant retains search, severity/source/type facets, pagination, details, and copy. It removes row selection, review tabs and review commands, delete commands, bulk commands, export, and all mutation dialogs.
- Target and global/scan vulnerability views remain the mutable owners. A Website scope reset clears selection and invalidates the current page token shape.

## Loading Contract

- 正常页面初始 loading 必须复用 `VulnerabilitiesDataTable`，通过 `loading` 和 `loadingRowCount` 交给共享表格渲染骨架行。
- `VulnerabilitiesVerticalView` 的 loading 与 resolved 表格必须传入同一个由
  `pageSize` 派生的 `stableSurfaceRowCount`；这是共享表格的首帧行节奏预留，不能用
  页面级 `min-h-*` 或伪造数据行替代。该预留只在 loading 时生效；解析后表格按真实
  行数或既有空态行收缩。
- `/vulnerabilities/` 的顶级 `VulnerabilitiesVerticalView` 使用共享自然表格
  流：页面负责正常纵向滚动，表头随表格行一起滚动，分页显示在带边框表格之外。
  保留已有的 `stableSurfaceRowCount` 首帧预留，但不要新增路由高度填充、固定表头
  或同框分页变体。
- `VulnerabilitiesDetailViewLoadingState` 必须接收完整 `VulnerabilitiesDetailViewState`，并把搜索、review tab、严重程度、来源、漏洞类型、排序、分页、列定义传给同一个表格 owner。
- `VulnerabilitiesVerticalView` 的 loading state 也必须复用当前 state 的 `VulnerabilitiesDataTable`。统计卡可以使用 `VulnerabilityStatCardsLoadingState`，但漏洞表格和 review tabs 不要回退到假列或假翻译 wrapper。
- `VulnerabilitiesVerticalView` 的 loading 与 resolved 分支必须把同一组 `vulnerabilities-table-toolbar`、`vulnerabilities-rows`、`vulnerabilities-pagination` `loadingSlots` 传给 `VulnerabilitiesDataTable`；共享表格在两个阶段各渲染一次这些区域，不能额外包一套仿表格骨架。
- `VulnerabilitiesDetailViewRouteFallback` 仅供目标/扫描详情 layout 自己的完整 shell loading 使用，不参与 Sidebar 软导航；它在同一个 sections owner 内驱动真实 `VulnerabilitiesDataTable` 的 `loading` 模式，不得引入独立仿表格文件或接回正常 query loading 主路径。
- 目标详情调用该 fallback 时使用稳定的两行首帧合同，并由目标 page 相同的 gutter 包裹；不要依赖首个可见 shell fallback 尚未取得的目标摘要，也不能回退为固定六行。
