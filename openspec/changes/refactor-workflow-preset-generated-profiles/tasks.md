## 1. 生成链改造（TDD）
- [ ] 1.1 为 preset 生成器新增行为测试：`default` 可从契约生成，先 RED
- [ ] 1.2 实现生成器最小能力（仅 `default`），并让测试 GREEN
- [ ] 1.3 为 profile overlay 合并新增测试（覆盖字段合法性、冲突、空 overlay）
- [ ] 1.4 实现 overlay 合并并保证 schema 校验通过后才写文件

## 2. 产物与命名约定
- [ ] 2.1 确定 preset 产物命名规则（含兼容 id 策略）
- [ ] 2.2 将生成目标接入 `server/internal/preset/presets`
- [ ] 2.3 增加回归测试：loader 可正确加载生成产物并保持去重/索引行为

## 3. 兼容与 API 回归
- [ ] 3.1 增加/更新 handler 级测试，锁定新 preset ID 与响应语义（breaking 后基线）
- [ ] 3.2 增加/更新 bootstrap 路由回归测试，确保管理 API 不回退
- [ ] 3.3 清理旧 preset 文件/兼容代码路径并更新对应测试
- [ ] 3.4 增加单 workflow 迁移测试：`subdomain_discovery` 生成产物可被扫描创建链消费
- [ ] 3.5 同步更新前端 preset 依赖（ID/文案/测试）到新基线

## 4. CI 门禁与开发流程
- [ ] 4.1 扩展生成脚本，将 preset 生成纳入统一入口
- [ ] 4.2 增加 CI 无差异检查，覆盖 preset 目录
- [ ] 4.3 更新开发文档：声明 preset 文件为生成产物，不建议手改

## 5. 验证
- [ ] 5.1 运行后端相关测试（catalog/preset/workflowschema/bootstrap）
- [ ] 5.2 运行 `openspec validate refactor-workflow-preset-generated-profiles --strict --no-interactive`
