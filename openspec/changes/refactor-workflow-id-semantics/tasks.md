## 1. Implementation
- [ ] 1.1 增加 manifest metadata 测试：缺失 `workflowId`、重复 `workflowId` 必须失败
- [ ] 1.2 实现 manifest `workflowId` 唯一性与格式加载校验
- [ ] 1.3 为持久化层补充 `workflow_name(s)` → `workflow_id(s)` 的迁移测试与回归用例
- [ ] 1.4 实现数据库列、repository 字段与 DTO 字段的 ID-first 迁移
- [ ] 1.5 将 catalog 查询接口与路由参数统一切换到 `workflowId`
- [ ] 1.6 落列表稳定排序规则 `workflowId ASC` 并补充回归测试
- [ ] 1.7 将 scan / runtime / worker 传递字段语义收敛为 ID-first
- [ ] 1.8 统一 seed、测试夹具、常量与文档中的 workflow ID 语义
- [ ] 1.9 运行相关测试与 OpenSpec 校验
