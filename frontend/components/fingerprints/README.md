# Fingerprint Workspace

该工作区只展示和管理 `fingerprinthub`。它不声明本地 aggregate 与任何外部执行器
的兼容性；该结论必须有上游源码证据。路由
`/tools/fingerprints/fingerprinthub/` 直接加载 `FingerPrintHubFingerprintView`，
复用 `FingerprintLibraryWorkspace`、`ContentHandoff`、共享业务表格和详情抽屉。

FingerprintHub 是共享 cursor-only pagination 的回归基线。当前已解析响应的
`nextPageToken` 是下一页的唯一依据，当前查询 token cache 中已访问的前页 token 是上一页的
唯一依据；搜索、筛选、排序或页大小变化必须清空 cache 并回到第一页。结果总数和页大小选择
继续保留，但不得显示页码、当前页、末页或由 `totalSize` 推导的总页数；首页按钮只清空 token 并重新请求第一页。

列表按名称搜索，按 `severity` 聚合筛选，并支持 FingerprintHub JSON 导入、当前
`fingerprinthub_web.json` 导出、批量删除和清空。浏览器只接收服务端字段化 DTO；
未知合法扩展仅在详情 `additionalFields` 中显示，不能访问持久化 payload。
