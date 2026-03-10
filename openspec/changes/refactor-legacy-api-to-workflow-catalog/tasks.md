## 1. API 收敛
- [ ] 1.1 新增 `/api/workflows` 与 `/api/workflows/:name` 只读接口（schema 驱动）
- [ ] 1.2 将 preset 路径迁移到 `/api/workflows/presets*`
- [ ] 1.3 删除旧目录路由注册与对应测试基线中的旧路径

## 2. 扫描合同改名
- [ ] 2.1 扫描创建请求/响应 DTO 从旧字段改为 `workflowNames`
- [ ] 2.2 handler/application/repository 全链路字段映射同步改名
- [ ] 2.3 更新错误文案与日志术语（统一为 workflow）

## 3. 遗留清理
- [ ] 3.1 删除 workflow 管理相关残留文件与 wiring 适配器
- [ ] 3.2 清理 migration、README、注释中的旧管理语义
- [ ] 3.3 清理不再使用的测试夹具与辅助函数

## 4. 验证
- [ ] 4.1 更新并通过路由回归测试（仅存在 `/api/workflows*`）
- [ ] 4.2 更新并通过扫描创建测试（仅使用 `workflowNames`）
- [ ] 4.3 执行 `go test ./...`（server）
- [ ] 4.4 执行 `openspec validate refactor-legacy-api-to-workflow-catalog --strict --no-interactive`
