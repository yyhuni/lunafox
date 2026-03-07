# Interface Naming Standards Implementation Plan

1. 为 middleware / logger / recovery / agent auth 补充表达新命名规则的失败测试。
2. 为 request / user / agent 上下文引入 accessor helper，并替换中间件内部的裸字符串存取。
3. 将 HTTP request / recovery 日志字段迁移到语义命名。
4. 将业务日志字段从 `snake_case` 迁移到点分命名。
5. 将剩余 `snake_case` `json` tag 迁移到 `camelCase`，排除数据库列名与第三方工具占位符。
6. 更新 `openspec/project.md` 的项目约定。
7. 运行受影响测试与 `openspec validate`，修正残留问题。

8. 为 gRPC 参数错误补测试并统一到 `scanId` / `targetId` / `itemsJson`。
9. 为 Loki / Prometheus labels 补约束测试，明确 `agent_id` / `container_name` 保持 `snake_case`。
10. 将显式命名的测试路由 path param 从 `:scan_id` 调整为 `:scanId`。
