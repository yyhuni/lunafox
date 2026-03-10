# Workflow Config Object Canonical Design

## Context
当前系统已经在 workflow 维度建立了 `schema / manifest / profile` 分层，并持续把默认值、preset 生成、ID 语义等能力收敛到更清晰的架构基线上。

但配置本身仍然保留“字符串 canonical”历史包袱：

- `profile.configuration` 是字符串，内部再嵌一段 YAML。
- catalog API 返回字符串配置。
- scan create 接收字符串配置，再在 server 内 parse 成对象。
- scan 和 scan_task 仍把 canonical 配置存成文本。
- 前端围绕 YAML 文本拼接、正则匹配 capability、直接覆盖字符串状态实现选择与编辑。

这会让 defaulting、overlay merge、schema validation 与 runtime boundary 无法围绕单一事实源协同演进。

## Goals / Non-Goals

### Goals
- 明确 workflow 配置在系统内的唯一 canonical 形态是结构化对象。
- 去掉 profile 文件中的 YAML-in-YAML 结构。
- 让 profile、catalog、scan create、scan persistence、frontend editor 都围绕对象模型工作。
- 将 YAML 限定为展示、导入导出、调试和必要协议边界的派生格式。
- 为 defaulting、overlay merge、schema validation 和后续配置迁移建立统一对象层入口。

### Non-Goals
- 不在本次同时重写 worker 内部 typed config 模型。
- 本次同时修改 runtime protobuf、agent runtime client 与 worker task config 读取链路，使跨进程边界也采用对象契约。
- 不在本次引入新的动态配置 DSL 或外部配置格式。
- 不在本次引入长期双写或长期双字段兼容策略。

## Approaches

### A. 仅修正 profile 文件形态
- 做法：把 `profile.configuration` 从字符串改为 YAML mapping，但 catalog API、scan API、DB 仍继续使用字符串。
- 优点：改动小，能消除最明显的 YAML-in-YAML。
- 缺点：系统整体仍以字符串为 canonical，不能解决扫描创建、前端编辑器、持久化与 merge/defaulting 的根因问题。

### B. 后端对象化，外部接口保留长期字符串兼容
- 做法：内部模型切到对象，但长期同时支持对象和 YAML 文本输入输出。
- 优点：迁移平滑。
- 缺点：长期维护两套语义，容易把“临时兼容层”演变为永久复杂度。

### C. 对象 canonical + 短期实现迁移缝合（推荐）
- 做法：把 profile、catalog、scan、persistence、frontend 全部收敛到对象 canonical；仅在实现阶段允许极短期适配层，最终不保留长期双格式契约。
- 优点：结构最干净，能从根上解决配置漂移、重复 parse 和文本拼接问题。
- 缺点：跨层改动较大，需要严格按 TDD 分阶段推进。

## Decision
采用 **C. 对象 canonical + 短期实现迁移缝合**。

## Decisions

### 1. Canonical 数据模型
- 系统内 workflow 配置的 canonical 形态定义为对象：`map[string]any`（或等价别名 `WorkflowConfig`）。
- 顶层键仍为 `workflowId`，其下挂载各 workflow 自身配置对象。
- profile loader、catalog facade、scan create、repository mapper、frontend state 都必须围绕该对象模型工作。

### 2. YAML 的职责边界
- YAML 仍然是 workflow 配置的人类可编辑格式，但不再是系统内主存储或主 API 契约。
- YAML 只允许出现在：
  - profile 工件文件本身（作为 YAML 文件语法），但 `configuration` 是 mapping 而不是二次字符串；
  - 前端文本编辑器视图；
  - 导入 / 导出工具。
- 任何内部业务逻辑不得以“先转成字符串再处理”为主路径。

### 3. Profile 工件结构
- profile 文件结构改为：

```yaml
id: subdomain_discovery.default
name: 子域名发现
description: 生成的默认预设
configuration:
  subdomain_discovery:
    recon:
      enabled: true
      tools:
        subfinder:
          enabled: true
          timeout-runtime: 3600
          threads-cli: 10
```

- `workflowIds` 不在 profile 文件内手工维护，而是由 loader 从 `configuration` 顶层键提取。
- generated profile 产物必须直接输出该结构，不再先 marshal 内部 YAML 再作为字符串字段写入外层 YAML。

### 4. API 与前端契约
- catalog profile API 的 `configuration` 返回 JSON object。
- scan create / quick scan / scheduled scan 的 `configuration` 输入改为 JSON object。
- scan detail 的 canonical 配置也返回 JSON object；若 UI 仍需 YAML 文本视图，应在前端本地完成对象 ⇄ YAML 的序列化与解析。
- 前端不得继续把配置状态作为字符串主状态保存；文本编辑器只是对象状态的一个派生视图。

### 5. Persistence 与 runtime boundary
- `scan` 表新增/切换为 `configuration JSONB` 作为 scan 级 canonical 配置。
- `scan_task` 表新增/切换为 `workflow_config JSONB` 作为 workflow-slice canonical 配置。
- repository / domain projection 均以对象字段为准。
- runtime protobuf `TaskAssign` 直接传递对象结构；agent 将对象写入 JSON task config 文件，worker 以对象方式读取。

### 6. Merge / defaulting / validation 顺序
- 统一顺序固定为：
  1. 读取 profile YAML 或 API JSON object
  2. 得到 canonical object root
  3. 执行 overlay merge / defaulting
  4. 对归一化后的对象执行 schema validation
  5. 持久化 canonical object
  6. 派生 workflow slice object
  7. 如边界需要，再序列化为 YAML
- `mergeWorkflowConfigurations`、capability 解析、workflow slice 提取都必须改成对象语义，不再依赖字符串拼接或正则。

### 7. Migration Strategy
- 实施层面允许短期迁移缝合，但最终 merge 基线不保留长期双格式契约。
- 推荐迁移顺序：
  1. profile + catalog 先对象化，锁定 loader / DTO 语义；
  2. scan create / persistence 改为 JSONB canonical，并保留 outbound YAML adapter；
  3. frontend 状态与编辑器改为对象 canonical；
  4. 删除 legacy `yaml_configuration` / `workflow_config_yaml` 主链路；
  5. 更新生成器、fixtures、文档与测试基线。

## Risks / Trade-offs
- 风险：改动面横跨 server、frontend、generator 与数据库。
  - 缓解：按 seam 做 TDD，优先锁定 profile/catalog，再推进 scan/persistence，最后切前端。
- 风险：活跃中的 preset/defaulting/id 变更可能存在交叉。
  - 缓解：在 tasks 中显式把这几个 change 作为实现前基线检查项，并在 branch 上统一 rebase。
- 风险：runtime 协议仍携带 YAML 文本，可能被误用为事实源。
  - 缓解：明确规范该字段仅是 outbound projection，并通过测试阻止回写或反向依赖。

## Testing Strategy
- backend：以 profile loader / validator、catalog handler、scan create、repository contract、runtime mapper 测试为主，全部按 TDD 执行。
- frontend：以 service/hook contract、`workflow-config` helper、scan dialog state 与 editor 行为测试为主，先写失败测试再改实现。
- migration：补充数据库模型契约与 mapper 测试，锁定 JSONB 列语义。
- verification：最终同时运行 OpenSpec 校验、相关 Go tests、相关 Vitest 测试。

## Final Recommendation
本次变更的核心不是“把 YAML 换成别的格式”，而是把 **对象** 设为唯一 canonical，把 **YAML** 限定为人类编辑与边界投影格式。

只有这样，workflow 配置才能真正与 preset 生成、defaulting、schema validation、前端编辑器和持久化体系对齐，而不是继续围绕字符串做局部修补。
