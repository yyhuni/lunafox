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
| Worker | 高 | 扫描任务执行（核心逻辑） |
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

## API 端点

```
POST   /api/auth/login          # 登录
POST   /api/auth/refresh        # 刷新 token
GET    /api/auth/me             # 当前用户

POST   /api/users               # 创建用户
GET    /api/users               # 用户列表
PUT    /api/users/password      # 修改密码

GET    /api/organizations       # 组织列表
POST   /api/organizations       # 创建组织
GET    /api/organizations/:id   # 获取组织
PUT    /api/organizations/:id   # 更新组织
DELETE /api/organizations/:id   # 删除组织

GET    /api/targets             # 目标列表
POST   /api/targets             # 创建目标
GET    /api/targets/:id         # 获取目标
PUT    /api/targets/:id         # 更新目标
DELETE /api/targets/:id         # 删除目标

GET    /api/workflows/profiles     # 工作流配置档案列表
GET    /api/workflows/profiles/:id # 工作流配置档案详情

GET    /api/scans/:id/logs      # 扫描日志（afterId 分页，参数: afterId, limit）
```

## 时间语义标准（UTC）

- 数据库存储统一使用 `TIMESTAMPTZ`。
- 数据库连接强制 `TimeZone=UTC`。
- 服务端时间写入统一使用 UTC（含 GORM `NowFunc`）。
- API / WebSocket 时间字段统一按 RFC3339Nano 输出（UTC，`Z`）。
- CSV 导出时间统一为 RFC3339Nano UTC。

示例：`2026-02-09T12:34:56.123456789Z`
