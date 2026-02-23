# 数据模型：数据库健康标准化接入

## 1）顶层响应模型

### `DatabaseHealthSnapshot`

- `status`: `online | degraded | offline | maintenance`
- `observedAt`: ISO 8601 时间字符串
- `role`: `primary | replica`
- `region`: 字符串（可选）
- `version`: 字符串
- `readOnly`: 布尔
- `uptimeSeconds`: 非负整数
- `coreSignals`: `CoreSignals`
- `optionalSignals`: `OptionalSignals`
- `unavailableSignals`: `UnavailableSignal[]`
- `alerts`: `HealthAlert[]`

约束：
- 顶层 `status` 由后端计算，前端不得覆盖。
- `observedAt` 必须为可解析时间戳。

## 2）核心信号模型

### `CoreSignals`

- `probeLatencyMs`: 非负数（核心探活延迟）
- `connectionsUsed`: 非负整数
- `connectionsMax`: 正整数
- `connectionUsagePercent`: `0..100` 浮点
- `lockWaitCount`: 非负整数（当前锁等待会话数）
- `deadlocks1h`: 非负浮点（按当前统计窗口折算的每小时死锁数）
- `longTransactionCount`: 非负整数（事务持续超过 60 秒的会话数）
- `oldestPendingTaskAgeSec`: 非负整数（最老待处理任务等待秒数）

状态判定建议：
- `offline`: 探活失败或核心依赖不可连接
- `degraded`: 可连接但核心信号超阈值或核心信号部分不可用
- `online`: 核心信号满足阈值
- `maintenance`: 维护模式显式标记

## 3）可选信号模型

### `OptionalSignals`

- `qps`: 非负数或 `null`
- `walGeneratedMb24h`: 非负数或 `null`
- `cacheHitRate`: `0..100` 浮点或 `null`

约束：
- 可选信号缺失不得单独触发 `offline`。

## 4）缺失信号模型

### `UnavailableSignal`

- `name`: 枚举字符串（如 `qps`、`cacheHitRate`、`lockWaitCount`、`oldestPendingTaskAgeSec`）
- `scope`: `core | optional`
- `reasonCode`: `permission_denied | timeout | unsupported | query_failed | unknown`
- `message`: 可选文本

## 5）告警模型

### `HealthAlert`

- `id`: 字符串
- `severity`: `info | warning | critical`
- `title`: 字符串
- `description`: 字符串
- `occurredAt`: ISO 8601 时间字符串

## 6）后端内部采集模型（实现参考）

### `DatabaseHealthProbe`

- `ok`: 布尔
- `latencyMs`: 浮点
- `error`: 可选错误文本

### `HealthEvaluationInput`

- `probe`: `DatabaseHealthProbe`
- `coreSignals`: `CoreSignals`
- `maintenanceFlag`: 布尔
- `thresholds`: 阈值配置集合

### `HealthEvaluationResult`

- `status`: 顶层状态
- `violations`: 命中的规则列表
- `unavailableSignals`: 缺失信号列表
