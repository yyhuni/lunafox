## Context
此前 workflow ID 已在 schema 层承担过可识别标识角色，但在新的上位架构下，schema 已经回归纯校验工件，workflow identity 必须迁移到独立 manifest。与此同时，业务代码与接口命名中仍有大量 `name` 语义残留，造成“机器标识”与“展示名称”混用。该问题横跨 catalog、scan create、runtime task assign、seed/tooling，多处语义不一致会持续放大维护成本。

## Goals / Non-Goals
- Goals:
  - 建立唯一、稳定、可验证的 workflow ID 契约。
  - 明确展示字段语义，避免与 ID 混用。
  - 让查询、编排、任务分发链路统一使用 `workflowId` 作为主键。
  - 让 manifest 成为 workflow identity 的事实来源。
  - 将持久化层命名也收敛到 ID-first。
- Non-Goals:
  - 不在本次引入 workflow 动态 CRUD。
  - 不在本次引入多语言展示名称体系。
  - 不在本次改变 workflow 执行逻辑本身。

## Approaches

### A. 语义文档化（最小改动）
- 做法：保留现有字段 / 命名（`name`, `workflowNames`），仅通过文档声明其本质是 ID。
- 优点：改动小、落地快。
- 缺点：代码可读性长期欠佳，语义歧义仍会持续制造新问题。

### B. 一次性 ID-first 收敛（推荐）
- 做法：内部接口、DTO、日志、测试命名统一改为 ID 语义；展示名单独字段承载；manifest `workflowId` 成为唯一事实来源；不保留兼容层；数据库列与持久化字段同步切换。
- 优点：语义最清晰，后续返工成本最低，契合 pre-launch 阶段。
- 缺点：改动面广，需要前后端、持久化层与测试基线同步切换。

### C. 双字段过渡（兼容期）
- 做法：一段时间内同时支持 `name` 与 `id` 两套命名，读新写双。
- 优点：迁移平滑。
- 缺点：兼容层复杂度高，且与“未上线一次性切换”的当前策略不一致。

## Decision
采用 **B. 一次性 ID-first 收敛**。

## Decisions
1. ID 主键来源
- workflow ID 以 manifest `workflowId` 为唯一事实来源。
- 启动时执行去重校验，重复或缺失视为配置错误并拒绝加载。
- workflow ID 规范化规则：
  - 格式：`^[a-z][a-z0-9_]{0,63}$`（小写 snake_case）
  - 不做自动规范化（不自动 lower、不自动替换字符），不合法即报错
  - 保留字：`all`、`default` 不可使用

2. 展示语义分离
- `displayName` 来源于 manifest `displayName`。
- `displayName` 不参与唯一性主键约束。
- 不再回退到 schema `title` 或旧的 `name` 字段。

3. 查询与路由语义
- 服务层查询接口统一为 `GetByWorkflowID`。
- HTTP 路由参数语义统一为 `:workflowId`。
- 列表接口返回顺序统一为 `workflowId ASC`，保证前端渲染稳定性。

4. 编排与运行链路
- scan create 入参、scan_task 持久化语义、runtime task assign 语义均以 workflow ID 为准。
- 工具链与测试常量统一为 ID 语义（如 `WORKFLOW_IDS`）。

5. 持久化命名同步切换
- 数据库列、repository 字段、domain 字段、DTO 字段同步从 `workflow_name(s)` 迁移到 `workflow_id(s)`。
- 不保留长期双列或双字段兼容。
- 若需要迁移脚本，必须与 repository / DTO 改动在同一变更中交付。

6. ID 不可变策略
- 已发布 workflow 的 `workflowId` 不允许原地修改。
- 如果业务必须改 ID，必须按“新增 ID + 迁移说明 + 旧 ID 下线”执行，不允许 silent rename。
- pre-launch 阶段可执行一次性切换，但仍需记录迁移映射以便排障与审计。

7. 字段命名收敛策略（ID-first）
- 跨层字段名优先表达“ID 语义”，避免 `Name` 歧义。
- 命名目标：
  - 单值：`WorkflowID`
  - 列表：`WorkflowIDs`
  - 展示：`DisplayName`

8. 与 schema / manifest 分离架构的关系
- schema 不再承担 workflow identity 事实源职责。
- manifest loader 负责 workflowId 的缺失、格式、去重与展示语义校验。
- 后续 contract naming 与 defaulting 变更必须建立在 manifest `workflowId` 基线上。

## Risks / Trade-offs
- 风险：字段重命名会影响前端、数据库与测试夹具。
  - 缓解：在 tasks 中明确“先 identity source，再 persistence migration，再跨层重命名”，分层改造并全链路回归。
- 风险：与进行中的 workflow 变更存在交叉。
  - 缓解：以 `refactor-workflow-schema-manifest-separation` 为上位基线，再让本变更与 naming / defaulting 提案统一 rebase。

## Migration Plan
1. 先落 manifest `workflowId` 唯一性与格式校验。
2. 再落数据库列与 repository / DTO 的 `workflow_id` 命名迁移。
3. 再落 catalog 查询接口命名与路由参数语义收敛。
4. 同步落列表稳定排序规则（`workflowId ASC`）并补充回归测试。
5. 再落 scan / runtime / worker 传递字段语义收敛。
6. 最后统一 seed、测试、文档与 OpenSpec 关联变更引用，并落迁移映射说明。
7. 迁移期间不引入长期兼容层；失败则整体回滚该变更分支。
