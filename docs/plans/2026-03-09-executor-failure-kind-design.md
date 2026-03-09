# Executor Failure Kind Refinement Design

**日期：** 2026-03-09

## 目标

提升 `agent/internal/task/executor.go` 中 `failureKind` 上报的可靠性，让失败分类优先基于执行路径而不是日志文案猜测。

## 设计方向

- 保持现有 `failed/cancelled/completed` 主状态不变
- 保持 `decode_config_failed` 这类已有细分类继续可用
- 将 `classifyFailureKind` 收敛为容器非零退出场景下的兜底细分
- 对已知失败路径直接显式赋值，减少对日志文本匹配的依赖

## 分类策略

- `worker_start_failed`：缺少 runtime socket、docker 不可用、生成 task token 失败、启动 worker 容器失败
- `container_wait_failed`：等待容器结束时返回非超时/非取消错误
- `task_timeout`：任务运行超时
- `container_exit_failed`：容器正常结束但退出码非 0 的默认分类
- `decode_config_failed`：仅在容器退出日志明确包含 workflow config 解码失败特征时细分
- `runtime_error`：保留为其他历史或未知运行错误的兜底枚举，不作为当前主要路径默认值

## 实现边界

- 仅修改 Agent 侧执行器与其测试
- 不修改 server 侧持久化逻辑
- 不做历史数据迁移
- 不扩展前端展示逻辑

## 涉及文件

- `agent/internal/task/executor.go`
- `agent/internal/task/executor_test.go`

## 验证方式

- `go test ./agent/internal/task -count=1`
- 重点验证 wait 失败、timeout、容器退出非零、decode config 日志分类
