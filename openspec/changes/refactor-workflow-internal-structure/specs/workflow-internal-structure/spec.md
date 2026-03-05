## ADDED Requirements

### Requirement: Workflow 内部能力目录 MUST 收敛到统一域路径
系统 MUST 将 workflow 相关内部能力统一收敛到 `server/internal/workflow` 下，以反映一致的领域语义与边界。

#### Scenario: 统一 schema/profile 包路径
- **WHEN** 开发者查看 server 侧 workflow 相关实现
- **THEN** schema 能力位于 `server/internal/workflow/schema`
- **AND** profile 能力位于 `server/internal/workflow/profile`
- **AND** 不再依赖历史顶层目录命名

#### Scenario: profile 产物目录统一
- **WHEN** workflow profile 产物被生成并落盘
- **THEN** 产物位于 `server/internal/workflow/profile/profiles`
- **AND** 不再使用 `server/internal/preset/presets`

### Requirement: Server 运行链路 MUST 只引用新 workflow 域路径
系统 MUST 在 bootstrap、catalog、scan 等链路中仅引用新路径包，避免目录语义回退。

#### Scenario: Scan schema gate 仅使用新 schema 包
- **WHEN** 扫描创建执行 workflow schema 校验
- **THEN** 代码引用 `server/internal/workflow/schema`
- **AND** 行为与迁移前一致（可识别 workflow、可执行 YAML 校验）

#### Scenario: Catalog profile 查询仅使用新 profile 包
- **WHEN** 目录接口读取 workflow profile 列表或单项
- **THEN** 代码引用 `server/internal/workflow/profile`
- **AND** 返回 profile 语义不变

### Requirement: Contract 生成链 MUST 使用新 schema 目录常量
系统 MUST 在 worker 生成脚本、Makefile 与文档中统一使用新的 server schema 输出目录。

#### Scenario: 生成脚本默认输出到新目录
- **WHEN** 执行 workflow contract 生成命令
- **THEN** server schema 产物写入新目录常量
- **AND** CI diff 检查覆盖新目录

### Requirement: 本次重构 MUST 不改变对外行为
系统 MUST 保持对外 API 返回语义、错误语义与 schema/profile 校验结果不变，仅进行内部目录与依赖路径收敛。

#### Scenario: 迁移前后外部行为一致
- **WHEN** 客户端调用现有 workflow 目录与扫描创建链路
- **THEN** 对外接口行为与迁移前一致
- **AND** schema/profile 校验结论不因目录迁移而变化

### Requirement: 历史路径引用 MUST 被清理
系统 MUST 清理 `server/internal/preset` 与 `server/internal/workflowschema` 的代码与文档引用，防止双路径长期共存。

#### Scenario: 仓库不存在旧路径依赖
- **WHEN** 执行全仓搜索旧路径字符串
- **THEN** 不存在运行代码或工具链中的旧路径引用
- **AND** 旧目录已删除
