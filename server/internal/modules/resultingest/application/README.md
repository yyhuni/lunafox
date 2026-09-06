# resultingest/application

通用规则请遵循：`server/docs/server-application-naming-template-v1.md`

resultingest 模块补充规则：

- **入口聚合**：`ResultIngestFacade.Ingest(ctx, ResultIngestCommand)` 是 agent task result batch 唯一应用边界，负责把已通过 transport scope 校验的结果批次物化为 scan evidence 和 asset inventory projection。
- **Facade 边界**：facade 内部的 closed registry 拥有 result type dispatch、逐项 typed decode、Server 复杂校验、原样结果项映射、effect accounting 与 materialization 错误语义；观测 URL、Host、证据和技术数组均按准入后的原始值传递。gRPC adapter 不选择 per-type handler，也不维护 result-kind-specific JSON parser。
- **服务编排**：当前同步物化支持 `asset.subdomain.v1`、`asset.host_port.v1`、`asset.website.v1`、`asset.website_technology.v1`、`asset.endpoint.v1`、`asset.directory.v1`、`asset.screenshot.v1` 与 `asset.vulnerability.v1`；新增 result type 必须先在 `contracts/results` 建立契约和校验，再在本模块补充 materialization path。
- **Nuclei 漏洞结果**：`asset.vulnerability.v1` 由 Nuclei Engine 以完整批次提交；registry 先校验 `source=nuclei`、URL/模板 ID、severity、CVSS 和 `rawOutput`，再通过 `VulnerabilitySnapshot.SaveResultBatchContext` 同库写入 Snapshot、Vulnerability、通知 occurrence 与统计栅栏。Snapshot 自然键是 `(scan_id,url,vuln_type,severity)`，当前漏洞自然键是 `(target_id,url,vuln_type,severity)`；重放只更新允许的观测字段并保留 `reviewed`。
- **adapter 例外**：当前无 transport-facing adapter service 例外；gRPC 只能依赖 `ResultIngestFacade` 或本模块声明的显式 application boundary interface。
- **内部 handler**：per-result materializer seam 均为 package-private，并统一接收原始 caller context；外部只能构造 facade，不能绕过 registry 直接调用某一种结果 handler。结果物化后的读模型刷新通过显式 summary updater port 表达。
- **一致性约束**：所有 Agent result type 都必须在同一同库事务内先重验 Target active、Scan/Task 归属、非终态和 Task Agent/Session/Epoch running lease，再写 Snapshot、Asset 和必要的 Scan summary。任一栅栏、projection 或 summary refresh 失败必须回滚整批并向上返回错误，不得转换为成功响应；transport 的早期 scope 校验只用于快速失败，不能替代该最终事务栅栏。
- Nuclei acknowledgement 与 Task terminal state 独立：acknowledged 后即使 Engine 后续取消、超时、非零退出或回滚，已经物化的 finding 仍可读，不执行终态补偿删除；只有未 acknowledgement 的批次才计入执行成功与否。
- **status-only 边界**：`Ingest` 要求完整 task/scan/target scope，在任何持久化效果前完成批级 envelope、closed-contract、复杂规则和最终 Target scope 准入。任一项无效、未授权或越界都失败整批，不能修复、过滤或产生 partial success。`ResultIngestOutcome` 仅供 Server 内部日志、指标和测试使用；成功响应始终为空，不能序列化、透传或按请求长度重建 summary/count/bool。
- **context、事务与重放**：同一个非 nil caller context 必须贯穿 decode、最终执行栅栏、snapshot durable write、asset projection 和同步 summary refresh；取消/超时必须返回错误。协调器创建 ambient transaction 后，所有参与同一批次的查询和写入都必须通过 `dbtx.Resolve(ctx, root).WithContext(ctx)` 取得该事务；不得使用 root DB、`context.Background()` 或嵌套事务，否则已锁定的 Scan/Target 会在第二连接上等待自身。可变观测在完整批次准入后按其定义的自然键收敛；Website v1 的 omitted optional 字段是完整观测的零值而非 patch mask。协调器事务失败不会留下 partial effects；重放同一 accepted batch 必须收敛到同一自然键状态。
- **新增 result type 准入**：必须先声明 Snapshot/Asset 稳定自然键，并标注 identity-only 或 complete-observation。后者还必须定义身份字段、核心产物、omitted 与合法空值语义、非法值拒绝与 latest-valid 选择；默认其 Snapshot、Asset、Scan summary 必须同库原子物化，并有重放收敛测试。仅已在 OpenSpec 明确批准的例外才可采用不同物化策略。缺失任一项不得加入 closed registry。

`asset.website_technology.v1` 是已批准的 current-only 例外：其 closed
`{url,tech}` batch 在任何写入前完成结构、复杂规则和 Task Target scope 校验；任何
无效或越界项都会拒绝整批。它只通过 dedicated Website technology upsert 创建缺失
Website 或替换现有 `Website.tech`；同一原始 URL 由最后一个提交项获胜，技术数组的
顺序、重复项和显式空数组均按原值保留。`tech:[]` 是可信零匹配，能够清空旧值。
该类型不写 Website Snapshot、Scan summary、finding、evidence 或 fingerprint
history，也不通过完整 `asset.website.v1` 物化路径。

Screenshot 是 complete-observation：`url` 是候选输入身份，WebP image 是必需核心产物，
`statusCode` 仅是 100--599 的可选证据。Decode 阶段拒绝未知/重复字段、不符合观测 URL
最低传输规则的值、非 canonical standard-base64 和非 WebP；materializer 只接收已通过 closed registry
的 item，原样原子写入 Screenshot Snapshot/Asset，不修改 Website projection，也不提供
额外的人工 Screenshot 上传兼容层。

Directory 是 complete-observation：每项必须且只能包含 `url`、`status`、
`contentLength`、`contentType`、`duration`。Registry 在写入前整批严格解码；
materializer 只从 URL authority 临时派生 Host 校验 authenticated Task Target scope，并以原始 URL
作为 Snapshot/Asset 自然键。任一无效、无法派生 Host 或越界项都会使整批在同库事务中回滚；一次
Server-backed gRPC OK 只在完整事务提交后产生。后续 Task/Scan 失败或取消不得补偿、隐藏或恢复旧值
覆盖已确认的 Directory 结果。
