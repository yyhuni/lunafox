# Backend 代码规范审查报告（逐条复核版）

**复核日期**: 2026-02-13  
**审查范围**: Server 端和 Worker 端代码结构、命名规范、边界规范  
**复核方法**: 逐条核对原报告中的每一个条目，结合当前代码、路由绑定方式、现行守卫脚本

## 1. 最终确认的问题（保留）

以下是复核后确认“确实有问题”的项。

### 1.1 P0（硬违规）

1. `server/internal/modules/agent/handler/agent_http_helpers.go`
- 结论: 保留（硬违规）
- 原因: 当前 `server/scripts/check-handler-boundaries.sh` 对 `agent/handler` 目录要求文件名必须是 `*_handler.go` 或 `*_mapper.go`。
- 当前命中报错: 该文件后缀 `_helpers.go` 不符合规则。

### 1.2 P1（业务/维护性问题，非硬违规）

1. `worker/internal/workflow/subdomain_discovery/subdomain_helpers.go`
- 结论: 保留（维护性问题）
- 原因: 单文件职责混杂（配置校验、命令构建、配置读取、文件统计、字符串清洗）。
- 建议: 最小拆分为 2 个文件（命令构建 + 配置/校验）。

### 1.3 P2（文档完善）

1. `server/internal/modules/agent/domain/README.md` 缺失
2. `server/internal/modules/agent/handler/README.md` 缺失

## 2. 已剔除的误报（不作为违规）

以下条目在原报告中被标为“违规”，复核后剔除：

1. `worker/internal/server/server_client_ports.go` 命名违规（误报）
2. `worker/internal/workflow/workflow_ports.go` 命名违规（误报）
3. “Handler 文件命名违规 34 个”（误报/规则外建议）
4. “Command/Query 文件缺少 `_handler.go` 后缀 17 个”（误报/规则外建议）
5. “Mapper 文件命名问题 7 个”（误报/规则外建议）
6. “snapshot/application 缺少 `_service.go` 后缀 7 个”（误报/规则外建议）
7. “Handler 100+ 顶层导出函数违规”（误报，方法被误判为顶层函数）
8. “Ports 组织 15 项违规”（大部分属于架构偏好，不是当前硬违规）
9. `catalog/application/worker_provider_config.go` 被定为“严重违规”（结论过重，降级为可选优化）

## 3. 关键复核依据（可复现）

### 3.1 守卫脚本复核

1. `bash server/scripts/check-naming-conventions.sh` -> 通过
2. `bash server/scripts/check-mapper-naming.sh` -> 通过
3. `bash server/scripts/check-router-boundaries.sh` -> 通过
4. `bash server/scripts/check-wiring-conventions.sh` -> 通过
5. `bash server/scripts/check-handler-boundaries.sh` -> 失败（仅 1 项：`agent_http_helpers.go`）
6. `cd worker && make check-naming` -> 通过

### 3.2 业务/项目逻辑复核要点

1. 报告中“把 handler 方法改为私有”的建议与当前路由组织冲突。路由在 `router` 包调用 handler 的导出方法，若改私有会破坏现有跨包调用。
2. 报告将“设计偏好”直接写成“违规”，例如把组合接口是否独立成 `*_store_ports.go` 视为硬性要求，这与当前项目规则不一致。

## 4. 原报告逐条复核清单（每一条都核对）

本节按原报告的每一条“问题条目”给出结论。

状态说明：
- `保留`：确认是问题
- `建议`：可改进但非违规
- `误报`：原结论不成立

### 4.1 Worker 端 3 条

| 条目 | 原结论 | 复核结论 | 状态 |
|---|---|---|---|
| `worker/internal/workflow/subdomain_discovery/subdomain_helpers.go` | 违规（helpers 后缀） | 文件命名本身不构成当前硬违规；但职责混杂，建议拆分 | 建议 |
| `worker/internal/server/server_client_ports.go` | 违规（ports 非资源化） | 已是资源化 `*_ports.go`，结论不成立 | 误报 |
| `worker/internal/workflow/workflow_ports.go` | 违规（ports 非资源化） | 已是资源化 `*_ports.go`，结论不成立 | 误报 |

### 4.2 Server 命名条目（原 2.2 ~ 2.4）

#### 4.2.1 泛名文件条目

| 条目 | 原结论 | 复核结论 | 状态 |
|---|---|---|---|
| `server/internal/modules/agent/handler/agent_http_helpers.go` | 违规（helpers 泛名） | 在 agent handler 目录确实违反当前命名守卫 | 保留 |

#### 4.2.2 “缺少 `_handler.go` 后缀（10 个）”逐项

| 文件 | 原结论 | 复核结论 | 状态 |
|---|---|---|---|
| `catalog/handler/engine.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `catalog/handler/preset.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `catalog/handler/target.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `catalog/handler/wordlist.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `catalog/handler/worker.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `identity/handler/auth.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `identity/handler/organization.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `identity/handler/user.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `security/handler/vulnerability.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `asset/handler/health.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |

#### 4.2.3 “Command/Query 文件缺少 `_handler.go`（17 个）”逐项

| 文件 | 原结论 | 复核结论 | 状态 |
|---|---|---|---|
| `scan/handler/scan_command.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `scan/handler/scan_query.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `scan/handler/scan_log_command.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `scan/handler/scan_log_query.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `scan/handler/worker_scan_query.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `identity/handler/organization_command.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `identity/handler/organization_query.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `identity/handler/organization_targets_command.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `identity/handler/organization_targets_query.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `snapshot/handler/directory_snapshot.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `snapshot/handler/endpoint_snapshot.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `snapshot/handler/host_port_snapshot.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `snapshot/handler/screenshot_snapshot.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `snapshot/handler/subdomain_snapshot.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `snapshot/handler/vulnerability_snapshot.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `snapshot/handler/website_snapshot.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |
| `catalog/handler/wordlist_content.go` | 应改 `_handler.go` | 当前项目允许；非硬违规 | 误报 |

#### 4.2.4 Mapper 命名条目（原报告写 7 个，实际只列出 3 个）

| 文件 | 原结论 | 复核结论 | 状态 |
|---|---|---|---|
| `scan/handler/scan_http_mapper.go` | `http_` 前缀冗余 | 可优化命名，但非违规 | 建议 |
| `scan/handler/scan_log_http_mapper.go` | `http_` 前缀冗余 | 可优化命名，但非违规 | 建议 |
| `snapshot/handler/snapshot_request_mappers.go` | 复数命名不规范 | 可优化命名，但非违规 | 建议 |

#### 4.2.5 Snapshot application 命名条目（7 个）

| 文件 | 原结论 | 复核结论 | 状态 |
|---|---|---|---|
| `snapshot/application/vulnerability_snapshot.go` | 应改 `_service.go` | 当前命名符合模块资源语义，非硬违规 | 误报 |
| `snapshot/application/subdomain_snapshot.go` | 应改 `_service.go` | 当前命名符合模块资源语义，非硬违规 | 误报 |
| `snapshot/application/host_port_snapshot.go` | 应改 `_service.go` | 当前命名符合模块资源语义，非硬违规 | 误报 |
| `snapshot/application/screenshot_snapshot.go` | 应改 `_service.go` | 当前命名符合模块资源语义，非硬违规 | 误报 |
| `snapshot/application/directory_snapshot.go` | 应改 `_service.go` | 当前命名符合模块资源语义，非硬违规 | 误报 |
| `snapshot/application/website_snapshot.go` | 应改 `_service.go` | 当前命名符合模块资源语义，非硬违规 | 误报 |
| `snapshot/application/endpoint_snapshot.go` | 应改 `_service.go` | 当前命名符合模块资源语义，非硬违规 | 误报 |

### 4.3 Handler 边界章节条目复核

| 原条目 | 原结论 | 复核结论 | 状态 |
|---|---|---|---|
| “所有 handler 文件都包含大量顶层导出函数” | 违规 | 误把方法当顶层函数；结论不成立 | 误报 |
| “Handler 文件只应包含 New*Handler” | 规范要求 | 与当前路由跨包绑定方式冲突，非现行规范 | 误报 |
| “违规文件数 25+、函数数 100+” | 统计结论 | 无可复现依据，且与守卫输出不符 | 误报 |
| “示例中 Login/Create/Handle 应私有” | 修复建议 | 私有化会破坏 router 包调用，建议不可直接采用 | 误报 |

### 4.4 Ports 组织章节条目复核（14 + 1）

#### 4.4.1 原 14 个“组合接口混合定义”逐项

| 文件 | 原结论 | 复核结论 | 状态 |
|---|---|---|---|
| `agent/application/agent_store_ports.go` | 混合定义违规 | 已在 `*_store_ports.go`，结构合理 | 误报 |
| `agent/application/registration_token_store_ports.go` | 混合定义违规 | 已在 `*_store_ports.go`，结构合理 | 误报 |
| `asset/application/directory_command_ports.go` | 混合定义违规 | 有优化空间（可拆 `DirectoryStore`），但非硬违规 | 建议 |
| `asset/application/endpoint_command_ports.go` | 混合定义违规 | 有优化空间（可拆 `EndpointStore`），但非硬违规 | 建议 |
| `asset/application/host_port_command_ports.go` | 混合定义违规 | 有优化空间（可拆 `HostPortStore`），但非硬违规 | 建议 |
| `asset/application/screenshot_command_ports.go` | 混合定义违规 | 有优化空间（可拆 `ScreenshotStore`），但非硬违规 | 建议 |
| `asset/application/subdomain_command_ports.go` | 混合定义违规 | 有优化空间（可拆 `SubdomainStore`），但非硬违规 | 建议 |
| `asset/application/website_command_ports.go` | 混合定义违规 | 有优化空间（可拆 `WebsiteStore`），但非硬违规 | 建议 |
| `catalog/application/organization_target_binding_ports.go` | 混合定义违规 | 单一端口文件，结论不成立 | 误报 |
| `catalog/application/wordlist_file_ports.go` | 混合定义违规 | 文件能力端口，结论不成立 | 误报 |
| `identity/application/auth_user_query_ports.go` | 混合定义违规 | 查询端口，结论不成立 | 误报 |
| `scan/application/task_runtime_store_ports.go` | 混合定义违规 | 已在 `*_store_ports.go`，结论不成立 | 误报 |
| `scan/application/task_store_ports.go` | 混合定义违规 | 已在 `*_store_ports.go`，结论不成立 | 误报 |
| `security/application/vulnerability_command_ports.go` | 混合定义违规 | 有优化空间（可拆 `VulnerabilityStore`），但非硬违规 | 建议 |

#### 4.4.2 原 1 个“严重违规”条目

| 文件 | 原结论 | 复核结论 | 状态 |
|---|---|---|---|
| `catalog/application/worker_provider_config.go` | 严重违规（Service + Store 同文件） | 架构可优化项，不应定性为严重违规 | 建议 |

### 4.5 Agent 结构评估条目复核

| 条目 | 原结论 | 复核结论 | 状态 |
|---|---|---|---|
| `application` 层 20 文件（过多） | 问题 | 当前 Go 文件计数为 19；“过多”属于主观评估 | 建议 |
| ports 文件过度分散（8 个，应合并为 2-3） | 高优先级问题 | 当前为 9 个；是否合并取决于团队偏好，非普遍更优 | 建议 |
| dto 目录应放 handler 内部 | 高优先级问题 | 与当前仓库通用模块结构冲突（各模块均为同级 `dto/`） | 误报 |
| 缺少 README（domain/handler） | 中优先级问题 | 事实成立，可改进 | 建议 |
| repository registration_token 应拆子目录 | 中优先级问题 | 可选重组，非必要 | 建议 |

## 5. 修订后的优先级建议

### P0（立即）

1. 处理 `server/internal/modules/agent/handler/agent_http_helpers.go` 命名问题（唯一硬违规）。

### P1（近期）

1. 拆分 `worker/internal/workflow/subdomain_discovery/subdomain_helpers.go`（维护性优化）。
2. 补齐 `agent/domain` 与 `agent/handler` README。

### P2（策略）

1. 讨论是否将以下“建议项”升级为强规范：
- handler 文件统一 `_handler.go`
- mapper 文件命名收敛
- 组合 Store 接口分离策略
2. 若升级，先改守卫脚本，再批量改代码。

