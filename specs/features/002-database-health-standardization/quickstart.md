# 快速验证：数据库健康标准化接入

## 前置条件

- Server 已启动，前端已启动。
- 已登录并持有有效 JWT（非 mock 模式）。
- 数据库实例可访问，且具备基础状态采集权限。

## 1）接口快速验证

```bash
curl -sS \
  -H "Authorization: Bearer <TOKEN>" \
  "http://localhost:8080/api/system/database-health"
```

预期返回示例：

```json
{
  "status": "online",
  "observedAt": "2026-02-23T12:00:00Z",
  "role": "primary",
  "region": "ap-southeast-1",
  "version": "PostgreSQL 15.4",
  "readOnly": false,
  "uptimeSeconds": 1048200,
  "coreSignals": {
    "probeLatencyMs": 18.5,
    "connectionsUsed": 42,
    "connectionsMax": 120,
    "connectionUsagePercent": 35.0,
    "lockWaitCount": 0,
    "deadlocks1h": 0,
    "longTransactionCount": 0,
    "oldestPendingTaskAgeSec": 45
  },
  "optionalSignals": {
    "qps": 512,
    "walGeneratedMb24h": 820,
    "cacheHitRate": 99.3
  },
  "unavailableSignals": [],
  "alerts": []
}
```

## 2）页面验证（P1）

1. 打开 `/zh/settings/database-health/`。
2. 验证页面顶层状态与接口 `status` 一致。
3. 验证核心指标展示与接口字段一致。
4. 验证时间字段可正常本地化显示（无 `Invalid Date`）。

## 3）故障与降级验证（P1/P2）

1. **探活失败场景**：
   - 模拟数据库不可达，验证状态为 `offline`。
2. **可选指标失败场景**：
   - 模拟可选指标查询失败，验证状态不被强制改为 `offline`，并出现 `unavailableSignals`。
3. **轮询失败场景**：
   - 在页面已有成功快照后让接口失败，验证页面显示“陈旧数据 + 错误提示”。
   - 首次加载即失败时验证显示“错误 + 重试”。

## 4）回归验证命令

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./...
cd /Users/yangyang/Desktop/lunafox/frontend && pnpm typecheck && pnpm test
```

## 5）验收清单

- 后端接口受 JWT 保护。
- 顶层状态由后端统一输出。
- 核心/可选信号分层生效。
- ISO 时间字段渲染正确。
- 故障态和陈旧态可见。
