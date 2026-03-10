# Change: 前端边界命名标准化

## Why
后端主线已经把 HTTP / JSON、错误字段、分页字段、显式 path param 收敛到 `camelCase`，但前端边界层仍保留了多种不标准做法：
- 部分 `frontend/services/*` 仍手写 `target_id`、`organization_id`、`page_size`。
- 部分 `frontend/types/*`、`frontend/hooks/*`、`frontend/lib/*` 中仍保留 `page_size`、`total_pages`、`created_at`、`is_read` 这类双轨兼容字段。
- 多个前端注释仍声称存在自动 `snake_case` / `camelCase` 转换，但当前仓库并没有真实启用这套转换链路。
- 同时，`proto`、数据库列名、GORM `column:`、SQL、外部 labels 这类边界本来就应保留各自生态的 `snake_case`，不应该被混入前端 HTTP 契约治理。

如果继续让前端边界层同时接受两套命名：
- 会模糊当前 Go 后端的实际契约；
- 会让 review 无法判断哪些 `snake_case` 是真实需要，哪些只是历史残留；
- 会使命名检查脚本要么过宽误伤，要么过松失效。

## What Changes
- 为前端 API 边界引入独立的命名标准化变更，明确它是对既有边界命名标准的第二阶段收敛。
- 定义前端 A / B / C 三类治理范围：
  - A 类：已确认对接当前 Go 后端的服务、DTO、响应解析逻辑，必须收敛到 `camelCase`
  - B 类：legacy / 占位 / 未确认对接当前主线的模块，先审计记录，不立即强改
  - C 类：非 API 边界代码，例如组件内部状态、localStorage key、测试计划 ID，不纳入本次治理
- 清理前端边界层对不存在的自动大小写转换的错误假设。
- 将命名检查脚本扩展到“已审计确认的前端边界文件清单”，而不是扫描整个 `frontend/`。
- 更新 `openspec/project.md`，把“前端 HTTP/JSON、proto、DB/SQL、日志、context”的命名标准总表写清楚。
- 明确 `proto`、generated、DB/GORM/SQL、Loki / Prometheus labels 等继续按各自生态标准处理，不纳入本次 HTTP 边界标准化。

## Impact
- Affected specs: `boundary-naming`
- Candidate audit scope:
  - `frontend/services/*`
  - `frontend/types/*`
  - `frontend/hooks/*`
  - `frontend/lib/*`
- Final implementation scope:
  - 以 `docs/plans/2026-03-06-frontend-boundary-naming-audit.md` 中确认的 A 类文件清单为准
- Supporting code and docs:
  - `scripts/ci/check-interface-naming.sh`
  - `.github/workflows/ci.yml`
  - `openspec/project.md`
  - `docs/plans/*frontend-boundary-naming*`
