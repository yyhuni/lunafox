# Old-Python Workflow Migration Playbook

本手册用于将 `old-python` workflow 批量迁移到当前 Go code-first 架构。

## 阶段 0：盘点

1. 列出待迁移 workflow 清单与优先级。
2. 记录每个 workflow 的：
- 输入配置项
- stage 顺序与并行关系
- 工具命令与关键参数
- 输出结果类型与落库路径

## 阶段 1：契约先行

1. 新建 `contract_definition.go`。
2. 完整迁移参数定义（key/type/required）。
3. 先生成 schema/docs/typed，确认和旧配置语义一致。

## 阶段 2：执行迁移

1. 将旧模板命令改为 `binary + args` 构建函数。
2. 按 stage 拆分执行函数并接入 `CmdRunner`。
3. 保持结果解析与保存接口不变，先保证行为对齐。

## 阶段 3：校验迁移

1. server 继续 schema gate。
2. worker 使用 strict typed decode + 业务 Validate。
3. 去除 workflow 内 runtime schema 依赖。

## 阶段 4：测试与门禁

1. 新增 workflow 单测（配置/阶段/命令构建）。
2. 跑一致性门禁：`make test-metadata`。
3. 跑全量回归：`go test ./worker/... -count=1` 与 `go test ./server/... -count=1`。

## 阶段 5：收尾

1. 删除旧模板残留与无用映射层。
2. 更新文档与示例 preset。
3. 按 workflow 维度分批合并，避免大爆炸提交。

## 建议节奏

- 每次迁移 1 个 workflow。
- 每个 workflow 走完 Red -> Green -> Refactor。
- 每完成 1 个 workflow 就执行一次一致性门禁。
