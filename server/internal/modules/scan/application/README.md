# scan/application

通用规则请遵循：`docs/server-application-naming-template-v1.md`

scan 模块补充规则：

- **入口聚合**：facade 按用例维度拆分（`facade_scan_create.go`、`facade_scan_query.go`、`facade_scan_lifecycle.go`、`facade_pull.go`、`facade_status.go`）。
- **服务编排**：create 主流程按阶段拆分（`create_normal_plan.go`、`create_normal_execute.go`、`create_normal_common.go`），runtime/lifecycle/log 独立编排（`task_runtime.go`、`lifecycle.go`、`scan_log.go`）。
- **端口拆分**：端口按 scan/task/runtime 边界拆分（`scan_*_ports.go`、`task_*_ports.go`、`task_runtime_*_ports.go`、`scan_create_target_lookup_ports.go`）。
- **模型命名**：输入输出模型按职责拆分（`scan_*_inputs.go`、`scan_query_outputs.go`、`scan_log_inputs.go`）。
- **历史迁移**：`aliases.go` 为历史聚合文件，后续新增优先资源化命名。
