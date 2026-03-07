# Interface Naming Enforcement Design

## 背景
边界命名标准已经在代码层完成一次性收敛，但如果没有自动化守门，后续在 review 中仍然会反复出现 `snake_case` JSON 字段、裸 Gin context key、历史日志字段和旧式 path param。

## 目标
- 把边界命名规则固化成仓库级检查脚本
- 把检查接入 GitHub Actions，作为持续门禁
- 允许数据库列名、Loki / Prometheus labels、第三方 provider 占位符等明确例外继续保留

## 方案
### 1. 检查入口
新增 `scripts/ci/check-interface-naming.sh`，默认扫描仓库内 `server` / `worker` 的相关源码。脚本既可本地直接运行，也作为 CI 中的唯一入口。

### 2. 检查维度
脚本聚焦已经完成治理、且误报可控的边界：
- `json` tag 中新增 `snake_case`
- `zap` 结构化日志字段使用裸 `snake_case`
- 跨包直接用裸字符串访问 `requestId` / `userClaims` / `agentId` / `agent` 这类 Gin context key
- 中间件 / gRPC 边界错误字段重新引入 `request_id` / `scan_id` / `target_id` / `items_json`
- 显式命名路由参数重新引入 `:scan_id` / `:target_id`

### 3. 例外策略
例外不单独抽成复杂配置文件，先以内嵌 allowlist 的方式固化在脚本里，来源直接对齐审计文档：
- `gorm:"column:..."`、SQL、migrations
- Loki / Prometheus labels
- 第三方 provider 占位符，例如 `api_key`
- legacy 反例测试断言
- generated 文件

### 4. CI 接入
直接接入 `.github/workflows/ci.yml`：
- 在 `detect-changes` 中增加 `interface_naming` 路径过滤
- 仅在 `server/**`、`worker/**`、`scripts/ci/**`、`.github/workflows/**`、相关 OpenSpec / 设计文档变更时触发
- 新增独立 job 执行 `bash scripts/ci/check-interface-naming.sh`

## 失败输出
脚本按规则逐项输出，命中时给出：
- 规则名称
- 命中的文件和行号
- 对应的修复提示

## 验证
- 先写脚本回归测试夹具，确认在“坏样例”上失败、在“好样例”上通过
- 再运行仓库真实扫描，确保当前主分支状态通过
- 最后运行 `openspec validate refactor-interface-naming-standards --strict --no-interactive`
