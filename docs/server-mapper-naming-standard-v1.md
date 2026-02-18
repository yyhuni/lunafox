# LunaFox Server Mapper 命名规范 v1

## 1. 目标

统一 `server/internal/modules/*` 的映射函数命名，避免 `Response` / `Output` 混用，降低跨模块阅读与重构成本。

本规范是对现有文档的补充，重点约束 **handler ↔ application ↔ dto** 的映射命名与放置位置。

---

## 2. 适用范围

- `server/internal/modules/*/handler`
- `server/internal/modules/*/application`
- `server/internal/bootstrap/wiring/*`

---

## 3. 命名规则（强制）

### 3.1 输入映射（HTTP DTO -> Application）

- 使用 `to*Input` 结尾
- 常见形式：
  - `toScanQueryInput`
  - `toScanCreateNormalInput`
  - `toSnapshotListQueryInput`
  - `toVulnerabilitySnapshotListQueryInput`

### 3.2 输出映射（Application/Domain -> HTTP DTO）

- 使用 `to*Output` 结尾
- 常见形式：
  - `toScanOutput`
  - `toScanDetailOutput`
  - `toScanStatisticsOutput`
  - `toWebsiteSnapshotOutput`
  - `toVulnerabilityOutput`

### 3.3 禁止命名

- 禁止新增 `to*Response`
- 禁止新增 `new*Response`
- 历史遗留命名在重构时必须迁移到 `*Output`

---

## 4. 放置规则（强制）

### 4.1 handler 层

- HTTP DTO 映射函数优先放在 handler 包内（可集中在 `*_http_mapper.go`）
- handler 负责请求绑定与 DTO 映射，不承载业务规则

### 4.2 application 层

- application 不直接依赖模块 `dto`
- application 对外输入输出模型使用 `*Input/*Output`（或清晰的 Query/Command 请求对象）

### 4.3 domain 层

- domain 不出现 HTTP DTO 概念与命名（`Request/Response`）

---

## 5. 文件命名建议（推荐）

- `scan/handler/scan_http_mapper.go`
- `scan/handler/scan_log_http_mapper.go`
- `snapshot/handler/snapshot_request_mappers.go`
- 若同一资源映射较多，可按资源拆分：`<resource>_http_mapper.go`

---

## 6. 迁移流程（推荐）

1. 新增 `to*Output` / `to*Input` 命名函数
2. 替换所有调用点
3. 删除兼容层（旧 `to*Response` 包装函数）
4. 运行 `gofmt` 与定向测试

---

## 7. 自检命令

在 `server/` 目录执行：

```bash
rg -n "to[A-Za-z0-9_]+Response\\(|new[A-Za-z0-9_]+Response\\(" internal/modules
```

若有命中，按本规范改为 `*Output`。

---

## 8. 示例（重构后）

- `security/handler/vulnerability.go`：`toVulnerabilityOutput`
- `catalog/handler/wordlist.go`：`toWordlistOutput`
- `identity/handler/organization.go`：`newOrganizationOutput`、`toTargetOutput`
- `snapshot/handler/*`：统一 `to*Output`（已移除旧 `to*Response` 兼容函数）

---

## 9. 与既有规范关系

- 本文不替代 `docs/repo-naming-standard-v2.md` 与 `docs/server-repository-standard-v1.md`
- 若规则冲突，以更严格的分层与边界规则为准
