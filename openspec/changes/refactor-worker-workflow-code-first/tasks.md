## 1. 执行模型收敛（TDD）
- [ ] 1.1 为当前 worker 可执行 workflows 补齐 code-first 执行路径行为测试（当前范围：`subdomain_discovery`）
- [ ] 1.2 运行针对性测试并确认 RED（先失败）
- [ ] 1.3 在 `worker/internal/activity` 实现标准化 `CmdRunner` 接口/组件（GREEN）
- [ ] 1.4 将命令执行从 `sh -c` 切换为 `binary + args[]` 接口并补齐安全回归测试
- [ ] 1.5 在 `CmdRunner` 中实现 `exec.LookPath(binary)` 启动前校验与明确错误返回
- [ ] 1.6 在 `CmdRunner` 中统一 stdout/stderr 捕获与截断安全日志策略
- [ ] 1.7 删除模板驱动命令构建与 key 映射执行路径
- [ ] 1.8 运行 workflow 相关测试，确认通过并无回归

## 2. 配置输入模型收敛（TDD）
- [ ] 2.1 为当前 worker 可执行 workflows 新增强类型 `Config` 解码与校验测试（当前范围：`subdomain_discovery`）
- [ ] 2.2 运行针对性测试并确认 RED
- [ ] 2.3 实现 YAML -> Config 解码与 `Validate()`，替换模板映射依赖（GREEN）
- [ ] 2.4 运行配置与扫描创建相关测试，确认 server 校验仍可工作

## 3. 契约与文档链路迁移
- [ ] 3.1 设计并实现从 Go 配置定义生成 schema/docs 的自动化代码生成命令
- [ ] 3.2 固化产物目录：schema 输出到 `server/internal/engineschema`，docs 输出到 `docs/config-reference`
- [ ] 3.3 定义 `contracts/gen/engineschema` 为可选镜像输出（默认关闭，仅对外发布时启用）
- [ ] 3.4 将当前 worker 可执行 workflows 的 schema/docs 来源切换到新机制（当前范围：`subdomain_discovery`）
- [ ] 3.5 增加一致性校验测试，确保文档、schema、运行时校验同源

## 4. 扩展机制基线
- [ ] 4.1 固化“新增 workflow = 新 Go 包 + 静态注册”开发模板
- [ ] 4.2 定义未来插件协议草案（进程隔离，不含 Go plugin in-process）
- [ ] 4.3 在文档中明确动态插件为后续能力，不纳入本次实现范围

## 5. 验证与发布准备
- [ ] 5.1 运行 `go test ./worker/... ./server/...`
- [ ] 5.2 运行契约生成命令并确认产物无手工改动
- [ ] 5.3 运行全工作流端到端回归（配置校验、任务编排、执行、结果写回）
- [ ] 5.4 运行上线前冒烟清单并确认仅存在 code-first 执行路径
- [ ] 5.5 运行 `openspec validate refactor-worker-workflow-code-first --strict --no-interactive`
