## 1. 基线与契约锁定（TDD）
- [x] 1.1 先为 profile loader / validator 增加失败测试，锁定 `configuration` 为对象 mapping，而不是字符串 YAML
- [x] 1.2 先为 catalog profile DTO / handler 增加失败测试，锁定 `configuration` API 契约为 JSON object
- [x] 1.3 先为 scan create 输入与 planner 增加失败测试，锁定对象 canonical 与 workflow slice 提取语义

## 2. 后端对象化收口（TDD）
- [x] 2.1 将 `server/internal/workflow/profile/*` 改为对象模型并保持 workflow ID 提取与 schema 校验通过
- [x] 2.2 将 catalog domain / DTO / adapter 改为对象配置返回，不再暴露字符串 canonical
- [x] 2.3 将 scan create / repository projection / runtime projection 改为对象 canonical，并保留必要的 outbound YAML adapter

## 3. 持久化切换（TDD）
- [x] 3.1 新增数据库迁移，把 `scan` / `scan_task` canonical 配置切到 JSONB 列
- [x] 3.2 增加 persistence model / mapper 测试，锁定 JSONB 列与对象字段语义
- [x] 3.3 清理 legacy `yaml_configuration` / `workflow_config_yaml` 作为事实源的依赖路径

## 4. 前端对象 canonical（TDD）
- [x] 4.1 为 workflow hooks / services 增加失败测试，锁定 profile 与 scan API 契约为对象配置
- [x] 4.2 为 `frontend/lib/workflow-config.ts` 增加失败测试，锁定 capability 解析、merge、YAML 视图转换都基于对象
- [x] 4.3 将 scan dialog / preset selector / config editor state 改为对象主状态，YAML 文本只作为派生视图

## 5. 生成器与文档收尾
- [x] 5.1 更新 `worker/cmd/workflow-contract-gen`，生成 mapping 形态的 profile 工件
- [x] 5.2 更新 profile fixture、文档和开发说明，明确 YAML 仅是文件语法，不再嵌第二段 YAML 字符串
- [x] 5.3 统一回写 OpenSpec / docs/plans 与相关测试基线

## 6. Runtime 契约对象化（TDD）
- [x] 6.1 更新 `proto/lunafox/runtime/v1/runtime.proto`，将 `TaskAssign.workflow_config_yaml` 切换为对象字段
- [x] 6.2 重生成 `contracts/gen/lunafox/runtime/v1/*` 并补齐 server runtime mapper 测试
- [x] 6.3 更新 agent runtime client / docker runner / worker config 读取链路，改为对象→JSON task config 文件

## 7. 验证
- [x] 7.1 运行相关 Go tests（profile / catalog / scan / runtime / agent / worker / contracts）
- [x] 7.2 运行相关 frontend Vitest（workflow hooks / helpers / scan state）
- [x] 7.3 运行 `openspec validate refactor-workflow-config-object-canonical --strict --no-interactive`
