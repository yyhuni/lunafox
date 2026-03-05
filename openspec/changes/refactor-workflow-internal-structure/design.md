## Context
当前 workflow 能力已经是“schema + profile”双源只读模型，但代码目录仍沿用历史命名：`preset`、`workflowschema`。这导致术语与边界不一致，并且在 worker 生成链、server import、文档中形成路径硬编码扩散。

## Goals / Non-Goals
- Goals:
  - 在 `server/internal/workflow` 下收敛 workflow 相关能力目录。
  - 统一命名为 `schema` 与 `profile`，对齐当前业务语义。
  - 一次性替换路径并删除旧目录，避免长期兼容负担。
- Non-Goals:
  - 不在本次引入新的对外 API 行为。
  - 不在本次改变 profile/schema 的业务规则。
  - 不在本次引入运行时动态生成 profile。

## Decisions
1. 目录收敛
- `server/internal/workflow/schema`：承载 schema 发现、metadata、校验。
- `server/internal/workflow/profile`：承载 profile 类型、loader、validator、service。
- `server/internal/workflow/profile/profiles`：承载 profile YAML 生成产物目录。

2. 无行为漂移约束
- 本次变更仅处理内部目录与依赖路径重构。
- 对外 API、错误语义、schema 校验结果必须与迁移前保持一致。

3. 兼容策略
- 采用 pre-launch 一次性切换，不保留旧目录兼容包。
- 所有 import/脚本/文档必须在同一变更内迁移完成。

4. 工程门禁
- 保留并强化“生成后无 diff”策略。
- 增加旧路径残留扫描，防止回退引用。
- 将本变更作为 `refactor-workflow-preset-generated-profiles` 的前置依赖，避免并发改动冲突。

## Path Mapping
| Old Path | New Path |
| --- | --- |
| `server/internal/workflowschema` | `server/internal/workflow/schema` |
| `server/internal/preset` | `server/internal/workflow/profile` |
| `server/internal/preset/presets` | `server/internal/workflow/profile/profiles` |
| `import .../internal/workflowschema` | `import .../internal/workflow/schema` |
| `import .../internal/preset` | `import .../internal/workflow/profile` |
| `worker scripts/Makefile: ../server/internal/workflowschema` | `../server/internal/workflow/schema` |
| `worker/profile output dir: ../server/internal/preset/presets` | `../server/internal/workflow/profile/profiles` |

## Risks / Trade-offs
- 风险：迁移跨度跨 server+worker+docs，容易漏改。
  - 缓解：按“迁移-编译-测试-全局搜索”顺序执行，最后做残留扫描。
- 风险：进行中的其他 change 可能继续引用旧路径。
  - 缓解：将本变更作为 workflow 相关 change 的前置基线，并在相关 tasks 标记依赖。

## Migration Plan
1. 复制并迁移 package 到 `internal/workflow/{schema,profile}`，保持 API 面一致。
2. 迁移 profile 产物目录到 `internal/workflow/profile/profiles`。
3. 替换 server import 并通过 catalog/scan/bootstrap 测试。
4. 替换 worker 脚本与 Makefile 默认路径并通过 worker 测试。
5. 更新 docs 路径说明与相关 OpenSpec 变更中的目录引用。
6. 删除旧目录与旧引用并执行全仓扫描。
