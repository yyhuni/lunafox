# Go Backend

Python Django 后端的 Go 重写版本。

## 技术栈

| 组件 | 选择 |
|------|------|
| Web 框架 | Gin |
| ORM | GORM |
| 配置 | Viper |
| 日志 | Zap |
| 认证 | JWT (bcrypt) |

## 迁移进度

### ✅ 已完成

| 模块 | 说明 |
|------|------|
| 项目基础 | 目录结构、配置管理、数据库连接、日志 |
| 数据模型 | 全部 33 个模型，含索引和约束 |
| JWT 认证 | 登录、刷新、中间件 |
| 用户 API | 创建、列表、修改密码 |
| 组织 API | 完整 CRUD（软删除） |
| 目标 API | 完整 CRUD（软删除、类型自动检测） |
| 工作流配置档案 API | 只读（列表/详情） |

### 🚧 待实现

| 模块 | 优先级 | 说明 |
|------|--------|------|
| Scan API | 高 | 扫描管理（发起、状态、结果） |
| Asset API | 高 | 资产查询（子域名、端口、漏洞等） |
| Agent / Engine Container | 高 | Agent 编排容器生命周期，Engine Container 执行具体扫描 |
| 定时任务 | 中 | 定时扫描 |
| 通知 | 低 | 扫描完成通知 |
| 统计 | 低 | 资产统计 |

### ⏳ 技术债务

| 项目 | 说明 | 优先级 |
|------|------|--------|
| Context 传递 | Repository/Service 加 context 参数 | 中 |
| 单元测试 | Handler/Service 层测试 | 中 |
| 接口抽象 | Repository 接口化（便于 mock） | 低 |
| 泛型重构 | 通用 Repository（等模块多了再做） | 低 |

## 运行

```bash
# 开发
make run

# 测试
make test

# 构建
make build
```

## 旧锁恢复

仅当已确认旧版结果入库锁链影响活动 Scan 时，遵循
[`docs/runbooks/scan-operations-old-lock-recovery.md`](../docs/runbooks/scan-operations-old-lock-recovery.md)。
全新安装、空库或无受影响 Scan 必须记录跳过，不得调用 Stop。

## Dev Compose 引擎安装

`docker/docker-compose.dev.yml` 会依次启动本地 OCI Registry、一次性
`engine-runtime-image-publisher`、`engine-package-publisher` 和
`engine-bootstrap`。Runtime Image publisher 先构建、发布并用宿主 Docker daemon
验证 immutable image refs；Package publisher 只消费其验证 receipt，生成
`.lfengine.tar.gz`、推送带 digest 的 OCI artifact 并生成开发 inventory。
bootstrap 使用与生产相同的拉取、签名、package contract 校验、digest 缓存和数据库
注册链；只有全部默认引擎注册成功后 Server 才启动，安装器随后才注册并启动 Agent。

生产和开发都以 `engine` 表作为已安装库存的事实源。`ENGINE_PACKAGE_CACHE_ROOT`
仅保存由 `package_digest` 定位的 archive 与展开缓存，不能用于发现、目录扫描或
替代 OCI 安装。

默认 scan workflow 定义的构建期源位于 `extensions/workflows/`。生产镜像会将它复制到 `/usr/local/share/lunafox/workflows`；开发 Compose 会以只读挂载提供同一路径。Server 仅从 `WORKFLOW_DEFINITIONS_ROOT` 读取和校验部署后的定义；该根目录缺失、不可读、定义无效或引用未安装引擎时启动失败，不会回退到源码目录。

## API 端点

> 以下列表反映当前 Go backend 的既有 HTTP route shape，用于本地导航与排障；它不是新的 API 风格规范。
> 新增或迁移 AIP-governed 边界时，以 `openspec/changes/standardize-project-boundary-contracts/` 为准：资源表示使用 canonical `name`，route param 从 generic `:id` 收敛到资源变量，更新目标形态使用 `PATCH` + `updateMask`，旧 `PUT` / `:id` / offset 分页属于迁移现状或例外。

```
POST   /v1/sessions          # 登录
POST   /v1/sessions:renew             # 刷新 token
GET    /v1/users/current             # 当前用户

POST   /v1/users               # 创建用户
GET    /v1/users               # 用户列表
PUT    /v1/users/password      # 修改密码

GET    /v1/organizations       # 组织列表
POST   /v1/organizations       # 创建组织
GET    /v1/organizations/:id   # 获取组织
PUT    /v1/organizations/:id   # 更新组织
DELETE /v1/organizations/:id   # 删除组织

GET    /v1/targets             # 目标列表
POST   /v1/targets             # 创建目标
GET    /v1/targets/:id         # 获取目标
PUT    /v1/targets/:id         # 更新目标
DELETE /v1/targets/:id         # 删除目标

GET    /v1/scanWorkflows                              # 扫描工作流列表（纯拓扑）
GET    /v1/scanWorkflows/:scanWorkflow                # 扫描工作流详情（纯拓扑）
GET    /v1/scanWorkflows/:scanWorkflow/profile        # 生成全关闭的 Profile 编辑草稿

POST   /v1/scans:batchCreate                          # 批量创建 Scan
POST   /v1/scans:quickCreate                          # 快速创建 Scan
GET    /v1/scheduledScans                             # Scheduled Scan 列表
GET    /v1/scheduledScans:summarize                   # Scheduled Scan UTC 管理概览
POST   /v1/scheduledScans                             # 创建 Scheduled Scan
POST   /v1/scheduledScans:batchUpdate                 # 原子批量更新 Scheduled Scan 启用状态
PATCH  /v1/scheduledScans/:scheduledScan              # 更新 Scheduled Scan（需 updateMask）

GET    /v1/scans/:scan/taskProgressLogs # 任务进度日志（AIP 分页，参数: pageSize, pageToken）

GET    /v1/admin/system/logEntries # Server/control-plane 日志（Loki-backed，参数: pageSize, pageToken, direction）
```

## 时间语义标准（UTC）

- 数据库存储统一使用 `TIMESTAMPTZ`。
- 数据库连接强制 `TimeZone=UTC`。
- 服务端时间写入统一使用 UTC（含 GORM `NowFunc`）。
- API / WebSocket 时间字段统一按 RFC3339Nano 输出（UTC，`Z`）。
- CSV 导出时间统一为 RFC3339Nano UTC。

示例：`2026-02-09T12:34:56.123456789Z`
