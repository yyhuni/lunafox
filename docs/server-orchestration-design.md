# 服务端编排（基础设计）

本文描述一个基础的服务端编排模型：服务端负责工作流调度和状态流转，agent/worker 仅执行单个任务。

## 目标
- 服务端负责任务调度、排序和阶段顺序。
- agent 只拉取 `pending`，worker 只执行并上报状态/结果。
- 支持阶段模式（串行/并行）与简单 fan-in/fan-out。
- 初始设计保持简单，可演进到 DAG。

## 非目标（初始版本）
- 完整的工作流 DSL 或可视化编排器。
- 高级重试策略与补偿步骤（后续可加）。
- 实时抢占或任务迁移。

## 当前痛点
- 任务拉取是全局的，无法按 scan 维度保证阶段顺序。

## 方案架构

### 高层流程
1. 服务端接收扫描请求并构建阶段任务列表。
2. 第一阶段任务置为 `pending`，后续阶段置为 `blocked`。
3. agent 拉取 `pending` 任务并分配给 worker。
4. worker 执行并上报状态与结果。
5. 编排器处理完成事件并解锁下一阶段任务。

### 数据模型
保留 `scan_task`，并为后续依赖关系预留扩展空间。

#### 表：`scan_task`（现有 + 扩展）
- `id`, `scan_id`, `stage`, `workflow_name`, `status`, `agent_id`, `config`, `error_message`, timestamps
- **新增**:
  - `priority` (int, default 0)
  - `attempt` (int, default 0)
  - `max_attempts` (int, default 1)
  - `stage_group` (int, default 0) -> 用于将多个 flow 归到同一阶段
  - `stage_mode` (enum: `sequential`, `parallel`) -> 可选，可从工作流定义推导
  - `blocked_reason` (varchar, optional)

建议状态集合：
- `blocked` -> 等待依赖
- `pending` -> 可调度
- `running` -> 已分配给 agent
- `completed` / `failed` / `cancelled` -> 终态

#### 表：`scan_task_dependency`（新增，可选后续）
- `task_id` (int)
- `depends_on_task_id` (int)
- `required` (bool, default true)

### 编排器职责
- 扫描创建时构建任务图（编译 workflow -> tasks + dependencies）。
- 维护任务就绪状态转换（`blocked` -> `pending`）。
- 校验依赖满足。
- fan-out（并行）与 fan-in（等待必需依赖完成）。

### 调度（拉取）规则
只返回 `pending` 的任务。
- 排序：`priority DESC`, `created_at ASC`。
- 通过 `FOR UPDATE SKIP LOCKED` 原子分配任务。

### 依赖满足逻辑
任务转为 `pending` 的条件：
- 所有 `required` 依赖必须进入终态。

任务完成时：
- 记录终态。
- 重新计算下游任务就绪性并解锁。

## 集成点

### 当前代码落位（2026-02）
- 扫描主链路已落到 `server/internal/modules/scan`：
  - `handler`：`scan.go`、`scan_log.go`、`task.go`
  - `service`：`scan.go`、`scan_create.go`、`scan_response.go`、`task_plan.go`、`pull.go`、`status.go`
  - `repository`：`scan.go`、`scan_command.go`、`scan_query.go`、`pull.go`、`status.go`、`stage.go`
- `asset` 模块不再承载 scan/scan-log 路由；scan 路由由 `server/internal/modules/scan/router` 统一注册。
- `bootstrap` 通过 `scanrouter.RegisterScanModuleRoutes` 接线扫描相关 API。
- 快照同步与 worker 读 scan 元数据统一依赖 `modules/scan/repository/ScanRepository`。


### 扫描创建
- 在 `CreateNormal` 中生成阶段任务。
- 第一阶段任务置 `pending`，后续阶段置 `blocked`。
- 当前不提供单独的“启动扫描”入口。

### 任务完成（UpdateStatus）
- 校验归属和状态流转。
- 持久化状态。
- 触发编排器解锁下游任务。

### Agent 拉取
- 只拉取 `pending` 任务。
- agent 内不做阶段顺序判断。

## 迁移计划（分阶段）

### Phase 0（立即安全）
- 在 SQL 中加入最小阶段门控，避免乱序执行。

### Phase 1（基础编排）
- 增加 `blocked` 状态。
- 在服务端实现阶段就绪判断。

### Phase 2（工作流归属）
- 工作流定义迁到 server 或共享模块。
- 服务端在创建扫描时生成任务列表。

## 待确认问题
- 工作流定义放在 server 还是共享模块？
- 是否需要每个 scan 的并发上限（超出 agent 能力以外）？

## 旧项目参考
旧项目定义了带“阶段模式”（串行/并行）的执行阶段，并列出每阶段的 flow。历史参考可在 tag `archive-old-before-removal-20260226` 中查看：

- `old/docs/scan-flow-architecture.md`

可映射到服务端编排：

- 旧项目的每个阶段对应 `stage_group`。
- `sequential` 表示下一个阶段必须等待当前组内任务全部进入终态。
- `parallel` 表示同组任务可同时解锁。

示例（对应旧结构）：

```yaml
stages:
  - mode: sequential
    flows: [subdomain_discovery, port_scan, site_scan, fingerprint_detect]
  - mode: parallel
    flows: [url_fetch, directory_scan]
  - mode: sequential
    flows: [screenshot]
  - mode: sequential
    flows: [vuln_scan]
```

服务端应将其编译为带依赖关系的 DAG，以确保阶段顺序与模式。

## 依赖生成规则（阶段到 DAG）
以 `stages` 定义为输入，生成 `scan_task` 与 `scan_task_dependency`（后续扩展）：

1. 为每个 flow 生成任务节点（`scan_task`）。
2. 设 `stage_group` 为阶段序号（从 1 开始）。
3. 生成跨阶段依赖：
   - 对第 N 阶段的每个任务，依赖第 N-1 阶段中“所有 required 任务”。
4. 阶段内依赖规则：
   - `parallel`：同阶段任务之间不互相依赖。
   - `sequential`：
     - 如果阶段内只有 1 个 flow，则无额外依赖。
     - 如果阶段内有多个 flow，可按 flow 顺序线性串联（flow[i] 依赖 flow[i-1]）。

说明：阶段内 `sequential` 是“同阶段串行”，跨阶段依赖仍然生效。

## 就绪判定算法
任务可从 `blocked` 变为 `pending` 需要满足：

1. 任务的所有 `required` 依赖进入终态。

伪代码：

```text
for each task in blocked:
  deps = dependencies(task)
  if all required deps are terminal:
     task.status = pending
```

## 状态机流转

核心状态流转：

```text
blocked -> pending -> running -> completed
                       \-> failed
                       \-> cancelled
```

说明：
- `cancelled` 用于用户主动停止扫描或系统取消。

## 调度与并发控制

- 调度只拉取 `pending`。
- 排序建议：`priority DESC`, `created_at ASC`。
- 并发控制建议放在两处：
  - agent 侧容量（已有 `max_tasks`）
  - server 侧 scan 级并发限制（可选扩展：每个 scan 同时 `running` 上限）
