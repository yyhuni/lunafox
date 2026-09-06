# Websites Components

网站列表属于标准 business-list 页面，表格外壳由 `WebSitesDataTable` 和共享 `BusinessListDataTable` 拥有。

网站表格和 relation evidence 都使用 cursor-only pagination：结果总数只作摘要，
当前响应的 `nextPageToken` 决定下一页，当前查询缓存中的已访问 token 决定上一页。
不得由总数推导总页数，也不得暴露页码、末页或当前页；首页按钮只清空 token 并重新请求第一页。

## Detail Inspection

- `WebsiteDetailDrawer` 是网站记录的只读检查入口，必须复用共享 `DetailDrawer` 的 shell、motion 和响应式约束；不要在本域直接组合 `SheetContent`。
- 网站表格的非交互区域通过共享行激活语义打开详情。复选框、链接、按钮和带 `data-row-click-exempt` 的后代必须保留自身操作，不能打开抽屉。
- 主表默认隐藏可由 URL 直接读取的主机名，以及响应头和响应体；三列仍可通过共享“显示列”菜单按需启用。详情抽屉按概览、技术栈、响应头、响应体纵向排列，响应头和响应体也可在各自本地滚动代码区中查看。
- URL 内容区以“HTTP 状态码 + URL”呈现，状态码不再占用独立列；状态码筛选仍由聚合筛选面板提供。
- 详情选择状态属于 `WebSitesView`，关闭抽屉不得改变当前筛选、排序、分页或行选择。

## Target Website Evidence Workspace

- `/targets/[id]/websites/` is the canonical target website collection and a reviewed controlled-list variant. It presents screenshot, website identity and technology, then screenshot-led, internally scrollable HTTP response information that grows with the row when needed. Scan-history websites remain a standard `WebSitesDataTable` business list.
- `TargetWebsiteEvidenceView` owns the canonical target website evidence-list table surface. The global Search Website read-only adapter reuses its presentational `RelationEvidenceListFrame`, `RelationEvidenceToolbar`, `WebsiteRelationEvidenceRow`, and screenshot preview components, but keeps Search query ownership and cursor pagination. It never imports target-scoped query, mutation, or bulk-management state.
- `useWebSitesViewState` remains the source of truth for the current website page, search, aggregate filters, pagination, selection, bulk add/delete, and CSV export. The Website List response supplies an optional Screenshot summary matched by the Screenshot natural `targetId + url` key; the summary never contains image bytes and the existing blob resolver still owns image loading.
- Website export behavior is unchanged: export-all uses the existing website export service, export-selected uses the existing website CSV fields, and neither path includes screenshot files, ZIP archives, manifests, or relation-specific fields.
- The wide evidence list orders website identity first, HTTP response second, and screenshot third beneath a shared 40px list header, then places the shared compact pagination as a natural-flow sibling outside the bordered records surface. The selection checkbox belongs to the website identity heading row, while selected-item deletion uses the shared bottom action bar and the existing confirmation dialog. It reuses shared search, facet, checkbox, action-menu and pagination owners.
- Each relation response panel uses the shared `ResponseEvidencePanel` owner, defaults to the response body, keeps body/header tabs inside the content region, and places the optional content-type value at the lower left and response-size value at the lower right; visually hidden terms preserve the definition-list semantics. Website relation evidence enables the shared metadata footer and supplies its own localized empty label; search cards use the same owner with location support but keep their response summary badges in the card header. Response content flexes within the screenshot-led row height instead of imposing a taller fixed row. Screenshot previews reuse the gallery's hover zoom and dark media-overlay treatment. Fingerprint summaries in both the evidence card and website-detail header measure available width and may use up to three rows before collapsing excess tags into a `+N` tag. The `+N` tag opens a shared dialog containing the complete technology set.
- Missing or unavailable screenshot enrichment renders an explicit empty screenshot state while retaining the website identity, HTTP information, resource controls and export.
- A website row is an inspection surface, not a row-level link. Its visible detail control opens `/targets/[id]/websites/[websiteId]/`; screenshot and external-link controls keep their own behavior.
- Website URLs are immutable observed strings. Display, copy and CSV/export paths use the stored value after only their outer transport escaping; website detail routes use `websiteId`, never a rebuilt URL. An external link passes that same string to the browser, but the browser's subsequent network serialization is outside the platform preservation guarantee.
- A website-detail return entered from the list restores page tokens, search and filters from session-scoped browser storage. Detail reads one resource through `GET /v1/websites/{website}`, validates that its canonical `targets/{target}/websites/{website}` name matches the route Target, and then keeps website identity above route-backed tabs for overview, IP addresses, URLs, directories and vulnerabilities.
- The four asset tabs directly embed the target-detail workspace owners with the current `targetId` and an explicit read-only `WebsiteAssetScope`. IP uses mandatory exact `host=="<website.host>"` and is presented only as domain-resolution/observed IP evidence, never IP ownership. URL, Directory, and Vulnerability use mandatory `websiteUrl=="<website.url>"`; this is a read-time virtual Scope that derives same scheme, hostname, effective port and path-segment boundary while ignoring query/fragment only for relation grouping. It never rewrites the stored URL, changes ordinary exact URL filters, participates in a natural key, or creates a relationship row. User filters may only narrow that range; scope changes reset pagination and selection.
- Website-scoped tabs retain search, filters, pagination, row details, and copy. They omit selection, export, create, delete, batch controls, confirmation dialogs, and vulnerability review commands. Target and scan workspace variants retain their existing management controls.
- Legacy `/targets/[id]/relations/` list and detail URLs only redirect to their website-route counterparts; they do not own a second collection or detail workspace.

## Loading Contract

- 正常页面初始 loading 必须复用 `WebSitesDataTable`，通过 `loading` 和 `loadingRowCount` 交给共享表格渲染骨架行。
- `WebSitesViewLoadingState` 必须接收完整 `WebSitesViewState`，并把搜索、筛选、排序、分页、列定义传给同一个表格 owner。
- `WebSitesViewRouteFallback` 仅供目标/扫描详情 layout 自己的完整 shell loading 使用，不参与 Sidebar 软导航；解析态 route page 与该 fallback 必须共同复用 `DetailAssetContentFrame`，并保留 `WebSitesDataTable` 的 `initial` loading presentation；普通 query loading state 不得再次嵌套该 frame。
- detail-layout loading state 不得提前显示可操作的搜索/筛选/显示列/导出/添加控件，也不得显示“共 0 条”或页码结论；网站查询成功后才进入真实表格或正式空状态。
- 不要在 detail-layout loading state 里复制 toolbar 按钮数、列数、分页或行高。真实表格结构变化时，loading 态应自动跟随真实表格。
- `WebsiteRelationEvidenceLoadingState` 必须通过 `RelationEvidenceListFrame` 复用目标站点页的工具栏动作位置、三栏列头、选择槽、行分隔和分页壳；骨架只替换行内内容与不可交互控件占位，不得另起一套列表结构。只读消费者可替换工具栏、footer 或行预算，但仍必须保留该 frame 作为唯一列表与分页层级。
- Identity 列要为 `WebsiteTechnologySummary` 已批准的三行指纹上界保留同一最小高度；该高度同时用于真实行和 loading 行，避免指纹数量在 handoff 时移动证据列表 footer。窄屏中实际内容如因换行更高，仍按自然高度扩展。
- 证据行的 HTTP 响应 loading 区必须使用共享
  `ResponseEvidencePanelLoadingState`。它会在宽屏随三栏行高拉伸，并在窄屏保持
  真实面板的自然高度；不得用 `h-52` 矩形代替它。
- 站点详情路由必须使用 `WebsiteRelationDetailLoadingState`，保留返回入口、站点身份和详情二级 Tab 的稳定几何；站点列表不得复用详情骨架，详情也不得回退到三栏列表骨架。解析完成后，嵌入的目标资产 workspace 必须继续拥有自己的 loading presentation。
- `TargetWebsiteEvidenceView` 与 `WebsiteRelationDetailView` 的查询级首屏加载必须留在各自的 `ContentHandoff` owner 内；详情壳尚未准备好时仍通过 `deferInitialSkeleton` 交由 route workspace owner 接管，避免同层重复骨架。两组 loading/resolved DOM 必须分别配对 `website-relation-evidence-surface` / `website-relation-evidence-list` 与 `website-relation-detail-surface` / `website-relation-detail-header` / `website-relation-detail-region` 的 `data-loading-slot`，供桌面和窄屏几何回归验证；详情身份头必须经由 `WebsiteRelationDetailIdentityFrame` 共享其断点、间距和双栏结构。
- 在 `/targets/[id]/websites/` 及其网站详情的冷启动中，`target-detail-shell` 是唯一可见的成对首屏 owner。两个站点子页面仍须作为 `section` expected owner 留在 route contract 中，但因 `deferInitialSkeleton` 会让它们在壳 handoff 前返回 `null`，必须显式排除首屏 paired-geometry；保留其本地 slots，供壳完成后的查询级或交互级 handoff 验证，不能借此删除或提前展示子页面 skeleton。
