## 1. 执行模型收敛（TDD）
- [x] 1.1 为当前 worker 可执行 workflows 补齐 code-first 执行路径行为测试（当前范围：`subdomain_discovery`）
- [x] 1.2 运行针对性测试并确认 RED（先失败）
- [x] 1.3 在 `worker/internal/activity` 实现标准化 `CmdRunner` 接口/组件（GREEN）
- [x] 1.4 将命令执行从 `sh -c` 切换为 `binary + args[]` 接口并补齐安全回归测试
- [x] 1.5 在 `CmdRunner` 中实现 `exec.LookPath(binary)` 启动前校验与明确错误返回
- [x] 1.6 在 `CmdRunner` 中统一 stdout/stderr 捕获与截断安全日志策略
- [x] 1.7 删除模板驱动命令构建与 key 映射执行路径
- [x] 1.8 运行 workflow 相关测试，确认通过并无回归

## 2. 配置输入模型收敛（TDD）
- [x] 2.1 为当前 worker 可执行 workflows 新增强类型 `Config` 解码与校验测试（当前范围：`subdomain_discovery`）
- [x] 2.2 运行针对性测试并确认 RED
- [x] 2.3 实现 YAML -> Config 解码与 Worker 侧权威 `Validate()`（含跨字段规则）（GREEN）
- [x] 2.4 固化 Server 侧 schema 仅做基础门禁（结构、类型、范围、required、unknown key）
- [x] 2.5 增加回归用例：允许出现“Server 通过但 Worker 拒绝”，并验证 fail-fast 错误语义
- [x] 2.6 实现统一错误响应结构 `{code, stage, field, message}` 并补齐单测
- [x] 2.7 落地错误码枚举：`SCHEMA_INVALID`、`WORKFLOW_CONFIG_INVALID`、`WORKFLOW_PREREQ_MISSING`
- [x] 2.8 落地配置版本字段：`apiVersion`（`v<major>`）与 `schemaVersion`（`MAJOR.MINOR.PATCH`）的必填与格式校验（Server schema gate）
- [x] 2.8.1 固化首发版本允许值枚举：`apiVersion in {v1}`、`schemaVersion in {1.0.0}`
- [x] 2.9 落地调度兼容门禁：按 `(workflow, apiVersion, schemaVersion)` 与 worker capability 精确匹配
- [x] 2.10 增加回归用例：版本不兼容返回 `WORKER_VERSION_INCOMPATIBLE`
- [x] 2.11 增加回归用例：版本字段缺失/格式非法返回 `SCHEMA_INVALID`
- [x] 2.11.1 增加回归用例：版本格式合法但不在首发枚举内返回 `SCHEMA_INVALID`
- [x] 2.12 运行配置与扫描创建相关测试，确认边界行为与文档一致

## 3. 契约与文档链路迁移
- [x] 3.1 设计并实现从 Go 配置定义生成 schema/docs 的自动化代码生成命令
- [x] 3.2 固化产物目录：schema 输出到 `server/internal/engineschema`，docs 输出到 `docs/config-reference`
- [x] 3.2.1 固化 schema 文件命名规则：`<workflow>-<apiVersion>-<schemaVersion>.schema.json`
- [x] 3.3 定义 `contracts/gen/engineschema` 为可选镜像输出（默认关闭，仅对外发布时启用）
- [x] 3.4 将当前 worker 可执行 workflows 的 schema/docs 来源切换到新机制（当前范围：`subdomain_discovery`）
- [x] 3.5 增加一致性校验测试，确保文档、schema、运行时校验同源

## 4. 扩展机制基线
- [x] 4.1 固化“新增 workflow = 新 Go 包 + 静态注册”开发模板
- [x] 4.2 定义未来插件协议草案（进程隔离，不含 Go plugin in-process）
- [x] 4.3 在文档中明确动态插件为后续能力，不纳入本次实现范围

## 5. 验证与发布准备
- [x] 5.1 运行 `go test ./worker/... ./server/...`
- [x] 5.2 运行契约生成命令并确认产物无手工改动
- [x] 5.2.1 校验生成产物命名符合版本化规则并与镜像输出同名
- [x] 5.3 运行全工作流端到端回归（配置校验、任务编排、执行、结果写回）
- [x] 5.4 运行上线前冒烟清单并确认仅存在 code-first 执行路径
- [x] 5.5 验证“Server 基础门禁 + Worker fail-fast”错误文案与错误码一致
- [x] 5.6 增加端到端用例覆盖三类错误码映射与文案
- [x] 5.7 增加端到端用例覆盖版本不兼容拦截与错误码映射
- [x] 5.8 增加端到端用例覆盖版本字段缺失/格式非法与 `SCHEMA_INVALID` 映射
- [x] 5.8.1 增加端到端用例覆盖“版本合法但超出首发枚举”与 `SCHEMA_INVALID` 映射
- [x] 5.9 运行 `openspec validate refactor-worker-workflow-code-first --strict --no-interactive`
