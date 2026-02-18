# LunaFox Server DDD 三期计划（执行版）

## 1. 目标

- 从“按技术分层 + GORM 模型即领域模型”推进到“按业务上下文 + 领域模型独立”。
- 保持现有 API 行为稳定，采用可回滚的渐进迁移。
- 先以 `scan` 模块打样，再复制到 `agent/asset/identity/catalog/snapshot`。

## 2. 领域边界（Bounded Context）

- `agent`：Agent 生命周期、注册、心跳、安装脚本。
- `scan`：扫描作业、任务编排、状态流转。
- `asset`：资产结果（网站、端点、目录、漏洞）查询与聚合视图。
- `identity`：用户、组织、权限及组织与目标关系。
- `catalog`：目标、引擎、字典、扫描配置元数据。
- `snapshot`：按扫描维度快照写入与查询。

## 3. 统一语言（Ubiquitous Language）

- `Scan`：一次扫描作业（聚合根）。
- `ScanTask`：扫描作业下的执行任务实体。
- `TaskPlan`：扫描任务阶段计划（值对象）。
- `TargetRef`：跨上下文读取目标信息的本地投影。
- `Snapshot`：按 `scan_id` 固化的历史视图。

## 4. Scan 聚合规则（必须由领域层维护）

- `pending -> running -> completed/failed/cancelled` 为主链路。
- `completed/failed/cancelled` 终态不可再进入 `running`。
- 只有 `pending/running` 才允许执行 stop（转 `cancelled`）。
- `failed` 必须携带错误信息。
- `target_id > 0` 且 `scan_mode` 必须为允许枚举。

## 5. 目标分层（scan 样板）

- `modules/scan/domain`：实体、聚合、值对象、领域错误、仓储端口。
- `modules/scan/application`：命令/查询用例，负责事务边界与跨仓储编排。
- `modules/scan/repository`：GORM 适配器（实现 domain 端口）。
- `modules/scan/handler`：HTTP 适配器，仅做输入输出映射。

## 6. 迁移顺序（建议）

1. 建立 `domain`（已开始）并冻结状态机规则。
2. 新增 `application` 用例服务，对外保持旧 handler 接口不变。
3. 让现有 service 逐步变薄，仅保留兼容门面后删除。
4. repository 拆为 `port + adapter`，禁止 domain 依赖 GORM。
5. 将跨模块读取改为 `Ref/Projection + ACL 映射`。

## 7. 完成判定（DoD）

- `scan` 上下文内：domain 不依赖 `gorm/gin/dto`。
- handler 不再直接操作 `model` 结构体。
- 业务规则由 domain 单测覆盖。
- `go test ./...` 全绿。

## 8. 当前进度（2026-02-06）

- 已建立 `scan/domain` 与 `scan/application` 骨架。
- 已落地 `Stop` 命令用例：`application.CommandService.StopScan`。
- 已落地 `Delete/BulkDelete` 生命周期用例：`application.LifecycleService`。
- 已落地 `CreateNormal` 用例：`application.ScanCreateService`。
- `scan/service` 主要负责错误映射与兼容门面。
- 已落地查询用例：`application.QueryService`（List/GetByID/GetStatistics）。
- 已清理 `service/task_plan.go` 重复实现，任务规划由 application 创建用例维护。
- 已开始迁移 `identity/organization`：新增 `application.OrganizationQueryService` 与 `application.OrganizationCommandService`，`identity/service/organization.go` 已改为门面。
- 已完成 `identity/user` 迁移：新增 `application.UserQueryService`、`application.UserCommandService`，`identity/service/user.go` 已改为门面并补充 application 单测。
- 已完成 `catalog/engine` 迁移：新增 `application.EngineQueryService`、`application.EngineCommandService`，`catalog/service/engine.go` 已改为门面并补充 application 单测。
- `catalog/handler/engine.go` 已补充 `ErrInvalidEngine` 映射，保持输入校验失败时的明确 400 语义。
- 已完成 `catalog/target` 迁移：新增 `application.TargetQueryService`、`application.TargetCommandService`，`catalog/service/target*.go`（含 detail/batch）已改为门面。
- `target` 详情汇总统计与批量创建（含组织关联）已下沉到 application，并补充 application 单测覆盖核心分支。
- 已完成 `catalog/wordlist` 迁移：新增 `application.WordlistQueryService`、`application.WordlistCommandService`，`catalog/service/wordlist*.go` 已门面化。
- `wordlist` 的文件校验、内容读写、文件统计同步与元数据更新已下沉至 application，并补充 application 单测。
- 已完成 `identity/auth` 迁移：新增 `application.AuthCommandService` 与认证端口，`identity/service/auth.go` 门面化。
- `identity/handler/auth.go` 已移除直连 DB/JWT，改为通过 `AuthService` 调用 application；`bootstrap/wiring.go` 完成注入改造。
- 已完成 `asset/website` 迁移：新增 `application.WebsiteQueryService`、`application.WebsiteCommandService`，`asset/service/website.go` 已门面化。
- `website` 的目标校验、批量创建/Upsert过滤、导出流与计数校验已下沉到 application，并补充 application 单测。
- 已完成 `asset/endpoint` 迁移：新增 `application.EndpointQueryService`、`application.EndpointCommandService`，`asset/service/endpoint.go` 已门面化。
- `endpoint` 的目标校验、批量创建/Upsert过滤、导出流与计数校验已下沉到 application，并补充 application 单测。
- 已完成 `asset/directory` 迁移：新增 `application.DirectoryQueryService`、`application.DirectoryCommandService`，`asset/service/directory.go` 已门面化。
- `directory` 的目标校验、批量创建/Upsert过滤、导出流与计数校验已下沉到 application，并补充 application 单测。
- 已完成 `asset/subdomain` 迁移：新增 `application.SubdomainQueryService`、`application.SubdomainCommandService`，`asset/service/subdomain.go` 已门面化。
- `subdomain` 的目标校验、域名类型校验、批量创建过滤、导出流与计数校验已下沉到 application，并补充 application 单测。
- 已完成 `asset/host_port` 迁移：新增 `application.HostPortQueryService`、`application.HostPortCommandService`，`asset/service/host_port.go` 已门面化。
- `host_port` 的目标校验、IP 聚合分页、导出流（含按 IP 过滤）、批量 Upsert/DeleteByIPs 已下沉到 application，并补充 application 单测。
- 已完成 `asset/screenshot` 迁移：新增 `application.ScreenshotQueryService`、`application.ScreenshotCommandService`，`asset/service/screenshot.go` 已门面化。
- `screenshot` 的目标校验、按目标分页查询、按 ID 查询、批量删除与批量 Upsert 过滤已下沉到 application，并补充 application 单测。
- 已完成 `asset/vulnerability` 迁移：新增 `application.VulnerabilityQueryService`、`application.VulnerabilityCommandService`，`asset/service/vulnerability.go` 已门面化。
- `vulnerability` 的目标校验、批量创建（含 severity 归一化与 raw_output 处理）、审阅状态流转、统计查询已下沉到 application，并补充 application 单测。
- 已完成 `snapshot/website_snapshot` 迁移：新增 `application.WebsiteSnapshotQueryService`、`application.WebsiteSnapshotCommandService`，`snapshot/service/website_snapshot.go` 已门面化。
- 已完成 `snapshot/directory_snapshot` 迁移：新增 `application.DirectorySnapshotQueryService`、`application.DirectorySnapshotCommandService`，`snapshot/service/directory_snapshot.go` 已门面化。
- `website_snapshot/directory_snapshot` 的 scan 校验、target 归属校验、快照入库与资产同步编排已下沉到 application，并补充 application 单测。
- 已完成 `snapshot/endpoint_snapshot` 迁移：新增 `application.EndpointSnapshotQueryService`、`application.EndpointSnapshotCommandService`，`snapshot/service/endpoint_snapshot.go` 已门面化。
- 已完成 `snapshot/subdomain_snapshot` 迁移：新增 `application.SubdomainSnapshotQueryService`、`application.SubdomainSnapshotCommandService`，`snapshot/service/subdomain_snapshot.go` 已门面化。
- `endpoint_snapshot/subdomain_snapshot` 的 scan 校验、target 归属校验、快照入库与资产同步编排已下沉到 application，并补充 application 单测。
- 已完成 `snapshot/host_port_snapshot` 迁移：新增 `application.HostPortSnapshotQueryService`、`application.HostPortSnapshotCommandService`，`snapshot/service/host_port_snapshot.go` 已门面化。
- 已完成 `snapshot/screenshot_snapshot` 迁移：新增 `application.ScreenshotSnapshotQueryService`、`application.ScreenshotSnapshotCommandService`，`snapshot/service/screenshot_snapshot.go` 已门面化。
- 已完成 `snapshot/vulnerability_snapshot` 迁移：新增 `application.VulnerabilitySnapshotQueryService`、`application.VulnerabilitySnapshotCommandService`，`snapshot/service/vulnerability_snapshot.go` 已门面化。
- `host_port/screenshot/vulnerability` 快照链路的 scan 校验、target 校验、快照入库与资产同步编排及漏洞数据校验规则已下沉到 application，并补充 application 单测。
- 已完成 `scan/service/scan.go` 结构化拆分：按职责拆为 `scan.go`（构造与依赖）、`scan_query.go`（查询口）、`scan_lifecycle.go`（生命周期口），保持行为与错误语义不变。
- `scan/service` 已满足单文件复杂度收敛目标（核心业务文件不再集中在单一超长文件），并通过 `scan` 模块与全量测试回归。
- 已完成 `catalog/application/target_command.go` 拆分：按职责拆为 `target_command_common.go`（契约与错误）、`target_command_crud.go`（单体 CRUD 用例）、`target_command_batch.go`（批量创建与组织绑定），单文件已降到阈值内。
- 已完成 `scan/application/create_normal.go` 拆分：按职责拆为 `create_normal_common.go`（契约与依赖）、`create_normal_execute.go`（创建主流程）、`create_normal_utils.go`（配置解析与引擎归一化）、`create_normal_plan.go`（阶段任务规划）。
- 本轮拆分后已通过 `go test ./internal/modules/catalog/... ./internal/modules/scan/...` 与 `go test ./...` 全量回归。
- 已完成 `scan/service` 目录收口：删除空壳 `task_plan.go`，并将响应转换函数并入 `scan_query.go`，目录内非测试文件数降至阈值内。
- 已完成 `scan/repository` 目录收口：将阶段推进逻辑并入 `status.go`，删除独立 `stage.go`，目录内非测试文件数降至阈值内。
- 已完成 `identity/application` 收口：`auth_types.go` 合并到 `auth_command.go`，减少跨文件跳转并降低目录碎片化。
- 当前 `service/application` 非测试文件已无超过 220 行项；剩余超目录阈值主要集中在 `snapshot/application`、`asset/application`，将按业务子域进一步下沉拆包。
- 已完成 `catalog/application` 目录收口：`wordlist_errors.go` 与 `wordlist_file_ops.go` 已并入 `wordlist_command.go` / `wordlist_query.go`，目录非测试文件数降至 8。
- 本轮结构审计后，`modules/*/{service,application}` 非测试文件行数已全部 <= 220。
- 当前仅剩目录文件数超阈值项：`asset/application`（14）、`snapshot/application`（16），下一步将按资源子域继续拆包或合并收口。
- 已完成 `asset/application` 目录收口：按“每个资源一个文件”合并为 `website.go`、`endpoint.go`、`directory.go`、`subdomain.go`、`host_port.go`、`screenshot.go`、`vulnerability_{query,command}.go`，目录非测试文件数降至 8。
- 已完成 `snapshot/application` 目录收口：按“每个快照资源一个文件”合并为 `website_snapshot.go`、`endpoint_snapshot.go`、`directory_snapshot.go`、`subdomain_snapshot.go`、`host_port_snapshot.go`、`screenshot_snapshot.go`、`vulnerability_snapshot.go`（含原 helpers），配合 `common.go` 目录非测试文件数降至 8。
- 本轮收口后，`modules` 下目录文件数（非测试）已无 >8 项，且 `service/application` 非测试文件已无 >220 行项。
- 本轮已通过 `go test ./internal/modules/asset/... ./internal/modules/snapshot/...` 与 `go test ./...` 全量回归。
- 已完成 `agent` 模块应用层落地：新增 `agent/application`（`QueryService` + `CommandService`），`agent/service` 已改为门面层（错误映射与兼容类型别名）。
- 已完成 `catalog/worker` 业务下沉：新增 `catalog/application/worker.go`，`catalog/service/worker.go` 仅保留门面职责；`target_command_common.go` 合并入 `target_command_crud.go` 以维持目录阈值。
- 已完成 `scan` 运行时链路下沉：新增 `scan/application/task_runtime.go`（任务下发与状态流转）与 `scan/application/scan_log.go`（日志查询/写入用例），`scan/service/{pull,status,scan_log}.go` 改为门面。
- 已完成 `scan/application` 收口：`create_normal_utils.go` 合并入 `create_normal_common.go`，保持目录阈值（非测试 <= 8）。
- 本轮后结构审计通过：`modules` 下目录非测试文件数无 >8，`service/application` 非测试文件无 >220 行，`domain` 包无基础设施污染导入。
- 本轮回归通过：`go test ./internal/modules/agent/... ./internal/modules/catalog/... ./internal/modules/scan/...` 与 `go test ./...`。
- 已完成全模块 `domain` 补齐：新增 `agent/domain`、`identity/domain`、`catalog/domain`、`asset/domain`、`snapshot/domain`，统一沉淀实体、错误与核心规则函数。
- 已完成跨层规则下沉：`agent/application` 注册与配置更新、`catalog/application` 目标识别与 worker provider 组装、`identity/application` 用户/组织规范化、`asset/snapshot` 漏洞 severity 与校验规则均改为调用对应 `domain`。
- 已完成门面层保持兼容：`service/handler` 外部契约未变，内部由 `application + domain` 驱动，维持现有 API 返回与错误映射。
- 当前阶段已满足“全模块具备 DDD 三层骨架（domain/application/repository）”目标，并通过全量回归。
- 本轮已完成 `identity` 纯化收口：`application` 层移除对 `identity/model` 与 `identity/repository` 的直接依赖，统一改为 `identity/domain` 实体与投影（含 `OrganizationWithTargetCount`）。
- 已新增 `identity/service/adapters.go` 作为 ACL 适配层：集中处理 `domain <-> model` 映射、组织目标关联错误映射（`repository.ErrTargetNotFound -> domain.ErrTargetNotFound`）。
- `identity/service/organization.go` 已移除对 `repository.OrganizationWithCount` 的外露依赖，改为服务内投影；`identity/handler/organization_targets.go` 已去除 repository 直连错误判断，统一通过 service 错误语义。
- 本轮改动后已通过：`cd server && go test ./internal/modules/identity/...` 与 `cd server && go test ./...`。
- 本轮继续推进 `catalog`：已完成 `engine` 链路纯化，`catalog/application/engine_{command,query}.go` 移除对 `catalog/model` 的直接依赖，改为 `catalog/domain.ScanEngine`。
- `catalog/service/engine.go` 已新增内嵌适配器（repo->domain / domain->model），对外返回契约保持不变；`engine` 应用层单测已同步迁移到 domain 类型。
- `catalog` 子模块回归通过：`cd server && go test ./internal/modules/catalog/...`。
- 本轮已完成 `catalog` 纯化收口：`catalog/application`（`engine/target/wordlist/worker`）已移除对 `catalog/model|catalog/repository|scan/model` 的直接依赖，统一改为 `catalog/domain` 实体、值对象与投影。
- 已补齐 `catalog/domain`：新增/扩展 `Target` 聚合字段、`Wordlist` 实体、`TargetAssetCounts/VulnerabilityCounts` 投影、`SubfinderProviderSettings` 与 `ProviderFormats`，并将 worker provider 配置规则统一下沉到 domain。
- 已完成 `catalog/service` 适配层：`target/wordlist/worker` 门面通过 adapter 映射 `repository <-> domain <-> model`，handler 对外契约保持不变。
- 结构约束修复：将 `identity/service/adapters.go` 拆分为 `adapters.go` 与 `organization_adapters.go`，确保 `service/application` 非测试文件无超过 220 行项。
- 本轮回归通过：`cd server && go test ./internal/modules/identity/... ./internal/modules/catalog/...` 与 `cd server && go test ./...`。
- 已完成 `snapshot` strict 收口：`snapshot/application` 已移除对 `snapshot/repository` 与 `snapshot/repository/persistence` 的直接依赖，相关 store/scan-lookup/asset-sync/raw-output 适配器统一迁移至 `bootstrap/wiring_snapshot_adapters.go`。
- `bootstrap/wiring.go` 已改为先组装 snapshot Query/Command service，再注入 Facade Service，确保 application 仅依赖端口与 domain 类型。
- `scripts/check-layer-dependencies.sh` 已将 `snapshot` 纳入 `STRICT_MODULES`，与 `agent/security/catalog/identity/asset/scan` 同级执行 application->repository 禁止约束。
- 当前阶段结论：在保持 HTTP API 行为不变前提下，DDD-Strict 分层收口（含 snapshot）已可由 `make check-architecture` 全自动守卫。
