# Frontend Boundary Naming Design

## 背景
LunaFox 后端主线已经把 HTTP / JSON 契约统一到 `camelCase`，但前端边界层仍留有历史兼容、误导注释和少量手写 `snake_case` 请求字段。这些做法既不符合当前后端契约，也会让命名守门脚本难以扩展。

## 目标
- 只治理前端 API 边界，不误伤整个前端仓库
- 让前端与当前 Go 后端契约在命名上单轨一致
- 明确哪些前端代码该改、哪些先不动、哪些根本不纳入

## 标准
### 应统一的边界
- A 类 `frontend/services/*`：对当前 Go 后端发请求的边界层，用 `camelCase`
- A 类 `frontend/types/*`：当前 API DTO，只保留 `camelCase`
- A 类 `frontend/hooks/*` / `frontend/lib/*`：分页、通知、错误解析等 API 边界逻辑，只接受 `camelCase`

### 不纳入本次治理的边界
- `proto` / generated
- DB / SQL / GORM
- Loki / Prometheus labels
- 组件内部状态、localStorage key、测试计划 ID、mock 业务值

## 守门前提
- 先产出 A / B / C 审计清单，再开始实施
- 所有自动化检查都必须以 A 类清单为准
- route 示例注释、接口说明字符串默认不纳入命名检查
- 已知高风险文件先挂到 B 类，例如 `frontend/services/worker.service.ts`

## 分类模型
### A 类：立即收敛
- 已确认对接当前 Go 后端主线
- 存在 `snake_case` 请求字段、兼容字段或误导注释
- 本轮会补测试并直接收敛

### B 类：先审计、不强改
- 无法确认是否仍接当前主线
- legacy / 占位模块，贸然修改风险高
- 先在审计文档中挂牌，暂不进入强校验

### C 类：不纳入
- 非 HTTP 契约边界代码
- 仅包含业务值、持久化 key、测试 ID 等

## 预期收益
- 前后端 HTTP 契约只保留 `camelCase` 一套命名
- 清理对不存在的自动转换机制的错误假设
- 让 CI 对前端边界做精确守门，而不是误伤整个 `frontend/`
