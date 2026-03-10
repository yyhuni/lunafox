## 1. Spec and contract
- [ ] 1.1 确认数据库健康 API 的响应契约（核心指标、可选指标、时间字段、告警字段）
- [ ] 1.2 在代码中对齐类型定义（前端 types 与后端 DTO）
- [ ] 1.3 确认状态语义（online/degraded/offline/maintenance）与阈值来源

## 2. Backend implementation
- [ ] 2.1 新增 `GET /api/system/database-health/` 路由并挂载到受保护分组
- [ ] 2.2 实现健康快照聚合服务（探活、连接、复制、备份、可选指标采集）
- [ ] 2.3 实现后端统一状态判定逻辑并输出 `unavailableSignals`
- [ ] 2.4 为采集超时、权限不足、部分失败增加降级处理
- [ ] 2.5 增加后端测试覆盖：正常、降级、离线

## 3. Frontend implementation
- [ ] 3.1 调整 `database-health` service/hook 与新 API 契约对齐
- [ ] 3.2 页面改为以后端状态为准，移除前端核心阈值判定
- [ ] 3.3 标准化时间展示：使用 ISO 时间字段进行本地化格式化
- [ ] 3.4 增加错误态与陈旧数据提示（请求失败时可见）
- [ ] 3.5 同步更新 mock 数据与真实契约一致

## 4. Validation
- [ ] 4.1 执行后端测试与关键架构守卫脚本
- [ ] 4.2 执行前端类型检查与相关测试
- [ ] 4.3 使用 `openspec validate update-database-health-standardization --strict --no-interactive` 校验变更工件
