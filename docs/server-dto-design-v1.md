# LunaFox Server DTO 重构设计 v1（确认版）

## 1. 决策输入（已确认）

本方案基于你已确认的 4 个选项：

- 维度：`A`（按资源类型拆分）
- 漏洞归属：`B`（独立 `security` 模块）
- 快照复用方式：`C`（独立 DTO + 转换层）
- 迁移方式：`B`（直接迁移到 `modules/*/dto`）

## 2. 背景与问题

当前 `server/internal/dto` 中存在明显混放：

- 同一资源分散在不同文件：例如 `Endpoint` 在 `asset_endpoints_content.go`，但部分快照在 `asset_discovery_vuln_and_extra_snapshots.go` 和 `snapshot_web_models.go`。
- 同一文件混合“资产 + 快照 + 漏洞”：命名与归属不一致，阅读成本高。
- `snapshot` 对 `asset` 有别名复用（如 `WebsiteSnapshotItem = WebsiteUpsertItem`），导致隐式耦合。
- 目前模块引用 `server/internal/dto` 的规模较大：`asset(21) / snapshot(15) / catalog(14) / scan(8) / identity(7) / agent(2)`，后续不治理会持续放大。

## 3. 目标与非目标

### 3.1 目标

- 按资源类型统一 DTO 归属，消除“按流程 + 按资源”混用。
- 模块私有 DTO 下沉到 `server/internal/modules/<module>/dto`。
- `vulnerability` 从 `asset` 拆为独立 `security` 上下文（DTO 先行，链路同步）。
- `snapshot` DTO 不再复用 `asset/security` DTO，改为转换层显式映射。
- 保持 API 路径与 JSON 字段兼容（前端无感）。

### 3.2 非目标

- 不改数据库表结构（仅模块与 DTO 归属调整）。
- 不改接口业务语义（状态码、字段、筛选逻辑保持一致）。

## 4. 设计原则

1. **单一归属**：每个 DTO 只属于一个模块。
2. **传输对象纯净**：DTO 不承载领域规则。
3. **跨模块只走转换**：禁止 `snapshot/dto` 直接 import `asset/security dto`。
4. **兼容优先**：路由和响应格式保持兼容，内部实现可重排。

## 5. 目标目录结构

```text
server/internal/modules/
  agent/
    dto/
      registration_models.go
      management_models.go
  identity/
    dto/
      organization_models.go
      user_models.go
      target_models.go
      common_models.go
  catalog/
    dto/
      target_models.go
      workflow_models.go
      wordlist_models.go
      worker_models.go
      preset_models.go
  scan/
    dto/
      scan_models.go
      scan_log_models.go
      task_models.go
  asset/
    dto/
      subdomain_models.go
      website_models.go
      endpoint_models.go
      directory_models.go
      host_port_models.go
      screenshot_models.go
  snapshot/
    dto/
      subdomain_snapshot_models.go
      website_snapshot_models.go
      endpoint_snapshot_models.go
      directory_snapshot_models.go
      host_port_snapshot_models.go
      screenshot_snapshot_models.go
      vulnerability_snapshot_models.go
  security/
    dto/
      vulnerability_models.go
```

共享层已收敛在 `server/internal/dto`：

- `core_http.go`（`BindJSON/BindQuery/BindURI`、统一错误响应）
- `pagination.go`（仅通用分页与 `PaginatedResponse`）

为避免 7 个模块重复维护，新增单一实现：

- `server/internal/modules/httpdto/http.go`（统一导出共享 HTTP DTO 能力）

模块侧统一约定：

- `modules/*/dto/models.go` 或 `modules/*/dto/*_models.go`：仅模块私有业务 DTO
- `modules/*/dto/common_http.go`：仅作为薄适配层，重导出 `modules/httpdto`
- 禁止在 DTO 模型文件（`models.go` 或 `*_models.go`）中使用 `server/internal/dto` 业务 DTO 类型别名

## 6. DTO 归属映射（从现状到目标）

### 6.1 资产与目录

- `Target* / TargetSummary / OrganizationBrief`：迁至 `modules/catalog/dto/target_models.go`
- `Website*`：迁至 `modules/asset/dto/website_models.go`
- `Subdomain*`（资产态）：迁至 `modules/asset/dto/subdomain_models.go`
- `Endpoint*`：迁至 `modules/asset/dto/endpoint_models.go`
- `Directory*`：迁至 `modules/asset/dto/directory_models.go`
- `HostPort*`：迁至 `modules/asset/dto/host_port_models.go`
- `Screenshot*`（资产态）：迁至 `modules/asset/dto/screenshot_models.go`

### 6.2 安全（独立模块）

- `Vulnerability*`（列表、创建、统计、批量审核）
  - 迁至 `modules/security/dto/vulnerability_models.go`

### 6.3 快照（完全独立）

- `SubdomainSnapshot*`：`modules/snapshot/dto/subdomain_snapshot_models.go`
- `WebsiteSnapshot*`：`modules/snapshot/dto/website_snapshot_models.go`
- `EndpointSnapshot*`：`modules/snapshot/dto/endpoint_snapshot_models.go`
- `DirectorySnapshot*`：`modules/snapshot/dto/directory_snapshot_models.go`
- `HostPortSnapshot*`：`modules/snapshot/dto/host_port_snapshot_models.go`
- `ScreenshotSnapshot*`：`modules/snapshot/dto/screenshot_snapshot_models.go`
- `VulnerabilitySnapshot*`：`modules/snapshot/dto/vulnerability_snapshot_models.go`

## 7. 关键依赖规则

### 7.1 允许的依赖

- `handler -> <module>/dto`
- `service/application -> <module>/dto`（仅入参出参）
- `snapshot/service -> asset/security dto`（仅在转换器中）

### 7.2 禁止的依赖

- `snapshot/dto -> asset/dto`
- `snapshot/dto -> security/dto`
- `asset/dto <-> security/dto` 相互 import

## 8. `snapshot` 转换层设计（落实 3C）

在 `server/internal/modules/snapshot/service` 新增（或整理）专用 mapper：

- `toAssetEndpointUpserts([]snapshotdto.EndpointSnapshotItem) []assetdto.EndpointUpsertItem`
- `toAssetDirectoryUpserts([]snapshotdto.DirectorySnapshotItem) []assetdto.DirectoryUpsertItem`
- `toAssetHostPorts([]snapshotdto.HostPortSnapshotItem) []assetdto.HostPortItem`
- `toSecurityVulnCreates([]snapshotdto.VulnerabilitySnapshotItem) []securitydto.VulnerabilityCreateItem`

约束：

- 转换逻辑只在 `snapshot/service` 集中维护。
- 不允许在 handler 内写字段拷贝逻辑。

## 9. 迁移执行计划（一步到位）

> 目标：单次迁移完成 `modules/*/dto` 收口，不保留模块 DTO 在 `server/internal/dto`。

### 步骤 1：建新 DTO 文件并放置类型

- 在各模块 `dto/` 下创建目标文件。
- 类型名、字段名、`json/form/binding` 标签保持不变。

### 步骤 2：全量改 import

- 按模块替换：
  - `asset/*` 改为 `modules/asset/dto`
  - `snapshot/*` 改为 `modules/snapshot/dto`
  - `catalog/*` 改为 `modules/catalog/dto`
  - `identity/*` 改为 `modules/identity/dto`
  - `scan/*` 改为 `modules/scan/dto`
  - `agent/*` 改为 `modules/agent/dto`
  - 漏洞链路改为 `modules/security/dto`

### 步骤 3：建立 `security` 漏洞链路最小闭环

- 新建 `modules/security/{dto,handler,service,repository,router}`（可最小化实现）。
- 将漏洞 handler/service/repository 从 `asset` 迁出到 `security`。
- 路由维持原 API 路径（例如 `/vulnerabilities`、`/targets/:id/vulnerabilities`）以保证兼容。

### 步骤 4：替换 snapshot 复用为转换层（已完成）

- 去掉别名复用（如 `type WebsiteSnapshotItem = WebsiteUpsertItem`）。
- 使用独立结构体 + service 层转换器。

当前状态（2026-02-07）：

- `snapshot/dto` 的 DTO 模型文件（`models.go` 或 `*_models.go`）已改为独立结构体定义，不再使用对 `asset/security` DTO 的别名。
- `snapshot/service/asset_sync_adapters.go` 已改为显式输出 `assetdto` 与 `securitydto`，完成跨模块转换层收口。

### 步骤 5：清理旧 DTO 文件（已完成）

- 删除 `server/internal/dto` 中模块私有 DTO。
- 共享层仅保留 `core_http.go` 与通用分页（`pagination.go`）。

当前状态（2026-02-07）：

- 各模块 DTO 模型文件（`dto/models.go` 或 `dto/*_models.go`）已改为独立结构体定义，不再使用 `server/internal/dto` 类型别名。
- 已删除 `server/internal/dto` 中历史业务 DTO 文件：
  - `agent_models.go`
  - `asset_endpoints_content.go`
  - `asset_subdomain_screenshot.go`
  - `asset_targets_web.go`
  - `scan_models.go`
  - `asset_discovery_vuln_and_extra_snapshots.go`
  - `snapshot_web_models.go`
- `middleware/agent_validation.go` 已切换使用 `modules/scan/dto.TaskStatusUpdateRequest`，不再依赖共享层业务 DTO。

### 步骤 6：回归验证

- `cd server && go test ./internal/modules/...`
- `cd server && go test ./...`
- `cd server && make check-ddd`（含 DDD + DTO 边界守卫）
- `cd server && make check-dto-selftest`（本地可选，验证守卫脚本可正确拦截违规）
- 冒烟检查：资产列表、快照入库、漏洞列表/统计/审核。

### 已完成项总览（截至 2026-02-07）

- 共享层已收敛：`server/internal/dto` 仅保留 `core_http.go` 与 `pagination.go`。
- 业务 DTO 已模块内聚：`agent/asset/catalog/identity/scan/snapshot/security` 均已迁移到 `modules/*/dto`。
- DTO 文件已按资源拆分：各模块统一采用 `*_models.go`（兼容少量 `models.go` 场景）。
- 漏洞链路已独立到 `security` 模块，`snapshot` 不再别名复用 `asset/security` DTO。
- 共享 HTTP 能力已单源化：`modules/httpdto` 统一导出，`modules/*/dto/common_http.go` 作为薄适配层。
- 守卫与验证已接入：`check-ddd`（含 DTO 边界守卫）+ `check-dto-selftest`（本地可选）+ 映射层单测。

## 10. 回滚点设计

- **RP1（安全）**：步骤 1 完成后，尚未删旧 DTO，可快速回退 import。
- **RP2（关键）**：`security` 路由接入前，漏洞仍在 `asset`，可回退到原 wiring。
- **RP3（终点）**：删除旧 DTO 前打 tag（或保留临时分支），失败可直接回滚提交。

## 11. 风险与缓解

- **风险：跨模块 import 循环**
  - 缓解：坚持“DTO 只入不出”，跨模块数据只走 service 转换器。
- **风险：漏洞拆分导致 wiring 大改**
  - 缓解：先保持 API path 不变，仅替换 handler/service 注入。
- **风险：snapshot 字段漂移**
  - 缓解：为 mapper 增加单测，确保字段一一对应。

## 12. 验收标准（DoD）

- `server/internal/dto` 不再承载模块私有 DTO。
- 每个模块只依赖自己的 DTO 包与共享 HTTP DTO。
- `snapshot` 不再出现对 `asset/security` DTO 的别名复用。
- 漏洞 DTO 与服务归属到 `security` 模块。
- 全量测试通过，前端接口兼容。

## 13. 建议的落地顺序（提交维度）

1. `dto` 新文件落位（不改行为）
2. import 全量切换
3. `security` 漏洞链路迁出
4. snapshot 转换层替换别名
5. 清理旧 DTO 与文档

## 14. 后续可选优化（不影响当前设计）

当前 `modules/httpdto` 已实现“共享 HTTP DTO 单源维护”。若后续希望语义更清晰，可将其迁移为 `server/internal/httpdto`，以强调其“横切基础设施”属性。

可选迁移路径（建议在独立提交中执行）：

1. 新建 `server/internal/httpdto`，复制 `modules/httpdto` 现有实现。
2. 将 `modules/*/dto/common_http.go` 的 import 从 `modules/httpdto` 切换到 `internal/httpdto`。
3. 保留 `modules/httpdto` 作为过渡 re-export（1 个版本窗口），减少并行分支冲突。
4. 确认无外部引用后再删除 `modules/httpdto`。

收益：

- 共享能力边界更清楚（业务模块 vs 横切能力）。
- 避免后续误解 `httpdto` 属于某个具体业务模块。

风险与控制：

- 风险较低（仅 import 路径迁移），但建议保持“先加后切再删”的三步节奏，降低冲突面。

---

如果确认本设计，可以按本文件直接进入实施，并先从“步骤 1 + 步骤 2”切第一批提交以降低冲突面。
