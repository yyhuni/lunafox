# Runtime Path Contract

## 目标
- 统一 server/agent/worker 对共享目录、socket、任务配置文件路径的约定。
- 避免路径常量散落在多端代码导致升级漂移。

## 单一来源
- 共享契约包：`contracts/runtimecontract/paths.go`

核心常量：
- `SharedDataRoot`: `/opt/lunafox`
- `DefaultWorkspaceRoot`: `/opt/lunafox/workspace`
- `DefaultResultsRoot`: `/opt/lunafox/results`
- `DefaultWordlistsRoot`: `/opt/lunafox/wordlists`
- `DefaultRuntimeMountPath`: `/run/lunafox`
- `WorkerRuntimeSocketName`: `worker-runtime.sock`
- `DefaultWorkerConfigPathEnv`: `CONFIG_PATH`

核心函数：
- `DefaultRuntimeSocketPath()`
- `BuildTaskWorkspaceDir(scanID, taskID)`
- `BuildTaskConfigPath(workspaceDir)`

## 约束
- 新增代码禁止直接硬编码 `/opt/lunafox` 或 `/run/lunafox/worker-runtime.sock`。
- 必须优先复用 `runtimecontract` 中的常量与构造函数。
- 测试可保留字面量断言，但业务路径拼接逻辑需走契约函数。
