# New Workflow Skeleton (Code-First)

本模板用于在 `worker/internal/workflow/<workflow_name>/` 新增内置 workflow。

## 必需文件

1. `workflow.go`
- 实现 `workflow.Workflow` 接口。
- 在 `init()` 中调用 `workflow.Register(workflow.Registration{...})`。
- 注册 `Name`、`Factory`、`ConfigDecoder`、`Contract`。

2. `contract_definition.go`
- 定义 `GetContractDefinition()`。
- 明确 `WorkflowName`、`APIVersion`、`SchemaVersion`。
- 声明 `Stages/Tools/Params` 契约。

3. `config_typed.go`
- 实现 strict decode（建议 `DisallowUnknownFields`）。
- 实现业务规则 `Validate()`（跨字段 fail-fast）。

4. `stages.go` + `stage_*.go`
- 保持 stage 执行代码可读、按职责拆分。
- 命令执行统一通过 `activity.CmdRunner`。

## 生成产物（自动）

1. `generated/<workflow>-<apiVersion>-<schemaVersion>.schema.json`
- 由全局生成入口 `make workflow-contracts-gen-all` 统一产出。

2. `config_typed_generated.go`
- 由生成器根据 `contract_definition.go` 自动产出，不要手改。

## 推荐测试最小集

1. 配置解码测试
- unknown key 拒绝
- required presence 拒绝
- 最小合法配置通过

2. 业务校验测试
- stage/tool 开关组合
- 关键必填字段

3. 契约一致性测试
- contract vs generated schema/docs/typed 一致
- stage metadata 与 contract 对齐

## 生成与回归命令

```bash
cd worker
make workflow-contracts-gen-all
make test-metadata
go test ./internal/workflow/<workflow_name> -count=1
go test ./... -count=1
```
