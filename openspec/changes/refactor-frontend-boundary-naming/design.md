# Design: 前端边界命名标准化

## Decision
采用“只治理前端 HTTP / JSON 边界，不扩散到整个前端仓库”的标准方案：

1. 前端 API 边界统一 `camelCase`
2. 不允许依赖不存在的全局自动 `snake_case` / `camelCase` 转换
3. 兼容治理按 A / B / C 三类分层处理，而不是一刀切扫描整个 `frontend`
4. `proto`、DB/GORM/SQL、外部 labels 继续按各自生态标准处理
5. 自动化守门只覆盖“已审计确认”的前端 A 类边界文件

## Scope
### In Scope
- 对前端边界候选范围做 A / B / C 审计分类
- A 类文件中的请求参数 / 请求体 / DTO / 分页解析 / 通知解析命名收敛
- `scripts/ci/check-interface-naming.sh` 的前端规则扩展
- 对应的前端测试、设计文档、审计文档与项目规范更新

### Candidate Audit Scope
- `frontend/services/*` 中直接构造 HTTP 请求参数 / 请求体的代码
- `frontend/types/*` 中直接表达后端 JSON 契约的 DTO
- `frontend/hooks/*` 与 `frontend/lib/*` 中直接解析 API 响应、分页元信息、通知负载的边界逻辑

### Out of Scope
- `proto` 与 `contracts/gen/**/*.pb.go`
- 数据库列名、SQL、GORM `column:`
- Loki / Prometheus labels
- 前端组件内部 state、localStorage key、测试计划 ID、mock 业务值
- 未确认接入当前 Go 后端主线的 legacy / 占位模块

## Classification Model
### A 类：立即收敛
满足以下条件之一的前端文件进入 A 类：
- 当前真实请求 `/api/*` 且后端主线已明确使用 `camelCase`
- 直接声明当前 API 响应 DTO
- 直接做分页、通知、错误负载等 API 响应解析

A 类代码必须：
- 请求参数使用 `camelCase`
- DTO 字段只保留 `camelCase`
- 只删除“经审计确认不再需要”的 `page_size`、`created_at`、`is_read` 等旧字段兼容
- 删除“自动转换”类误导注释

### B 类：先记录，不立即强改
满足以下任一条件的文件进入 B 类：
- legacy / 占位实现，无法确认是否对应当前后端主线
- 可能对接历史后端、第三方后端或未启用模块
- 修改后存在较高破坏风险但缺少当前契约证据

B 类代码必须：
- 被审计文档显式列出
- 在自动化检查中先排除
- 具备后续退出条件，例如“后端路由确认存在后再纳入 A 类”

### C 类：不纳入
- 非 HTTP 契约边界的前端代码
- 仅承载业务值而非字段名的字符串
- 开发工具配置、测试计划 ID、持久化 key 等

### Initial B-Class Hold Candidates
在当前已知事实下，以下文件默认先作为 B 类候选：
- `frontend/services/worker.service.ts`：尚未确认存在当前 Go 后端 `/workers` 主线路由，不进入首轮强改或强校验

## Naming Rules
### Frontend request / response DTO
- 使用 `pageSize`、`totalPages`、`targetId`、`organizationId`、`createdAt`、`isRead`
- 不新增 `page_size`、`total_pages`、`target_id`、`created_at`、`is_read`

### Frontend comments / docs
- 不允许保留“interceptor will convert to snake_case”这类错误注释
- 若存在兼容逻辑，必须明确写出兼容边界、保留原因和移除条件

### Proto / generated contracts
- `.proto` 字段继续使用 protobuf 标准 `snake_case`
- `contracts/gen/**/*.pb.go` 继续视为 generated 文件，不纳入 HTTP 命名规则

## Migration Strategy
### Step 1: 审计
- 生成前端 A / B / C 审计清单
- 给出每类文件的证据、原因和是否纳入首轮实施
- 产出 A 类精确文件清单，作为唯一实施范围

### Step 2: TDD 收敛 A 类
- 先补失败测试，表达“只接受 `camelCase`”的新契约
- 再清理服务层、DTO、分页解析、通知解析中的旧兼容字段
- 对 B 类保持只记录、不强改

### Step 3: 守门
- 扩展 `scripts/ci/check-interface-naming.sh`
- 仅检查 A 类文件清单 / 文件模式
- 继续排除 B 类、C 类、`proto`、generated、DB / SQL / labels

## Automation Guardrails
- 前端命名检查只对 A 类文件生效，不对整个 `frontend/` 生效
- 前端命名检查必须基于代码结构模式，而不是自由文本匹配
- 默认不因注释、接口说明字符串、route 示例占位符（如 `{target_id}`）而失败
- B 类文件必须显式列入 allowlist，并在审计文档中给出原因

## Verification
- `cd frontend && pnpm test -- --run <经审计确认的受影响测试>`
- `bash scripts/ci/check-interface-naming.sh`
- `openspec validate refactor-frontend-boundary-naming --strict --no-interactive`
