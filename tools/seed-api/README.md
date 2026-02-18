# API-Based Seed Data Generator

基于 API 的种子数据生成器，通过调用 Go 后端的 REST API 来创建测试数据。

## 功能特性

- ✅ 通过 HTTP API 创建测试数据（不直接操作数据库）
- ✅ 支持生成组织、目标、资产（Website、Subdomain、Endpoint 等）
- ✅ 自动认证管理（JWT token 自动刷新）
- ✅ 智能错误处理和重试机制
- ✅ 实时进度显示和统计
- ✅ 独立 JSON 构造，能发现 API 序列化问题

## 依赖要求

- Python 3.8+
- requests 库

## 安装

```bash
# 进入工具目录
cd tools/seed-api

# 安装依赖
pip install -r requirements.txt
```

## 使用方法

### 基本用法

```bash
# 生成默认数量的测试数据（15 个组织，每个组织 15 个目标）
python seed_generator.py
```

### 命令行参数

```bash
python seed_generator.py [OPTIONS]

Options:
  --api-url URL          API 地址 (默认: http://localhost:8080)
  --username USER        用户名 (默认: admin)
  --password PASS        密码 (默认: admin)
  --orgs N               组织数量 (默认: 15)
  --targets-per-org N    每个组织的目标数量 (默认: 15)
  --assets-per-target N  每个目标的资产数量 (默认: 15)
  --clear                清空现有数据
  --batch-size N         批量操作的批次大小 (默认: 100)
  --verbose              显示详细日志
  --help                 显示帮助信息
```

### 使用示例

```bash
# 1. 启动 Go 后端（另一个终端）
cd ../../server
make run

# 2. 生成小规模测试数据
python seed_generator.py --orgs 5 --targets-per-org 10

# 3. 生成大规模测试数据
python seed_generator.py --orgs 50 --targets-per-org 20

# 4. 清空数据后重新生成
python seed_generator.py --clear --orgs 10

# 5. 使用自定义 API 地址
python seed_generator.py --api-url http://192.168.1.100:8080

# 6. 显示详细日志
python seed_generator.py --verbose
```

## 项目结构

```
tools/seed-api/
├── seed_generator.py      # 主程序入口
├── api_client.py          # API 客户端（HTTP 请求、认证管理）
├── data_generator.py      # 数据生成器（生成随机测试数据）
├── progress.py            # 进度跟踪（显示进度和统计）
├── error_handler.py       # 错误处理（重试逻辑、错误日志）
├── requirements.txt       # Python 依赖
└── README.md              # 使用说明
```

## 生成的数据

### 组织 (Organizations)
- 随机组织名称和描述
- 每个组织关联指定数量的目标

### 目标 (Targets)
- 域名（70%）：格式 `{env}.{company}-{suffix}.{tld}`
- IP 地址（20%）：随机合法 IPv4
- CIDR 网段（10%）：随机 /8、/16、/24

### 资产 (Assets)
每个目标生成以下资产：
- **Website**: Web 应用（URL、标题、状态码、技术栈等）
- **Subdomain**: 子域名（仅域名类型目标）
- **Endpoint**: API 端点（URL、状态码、匹配的 GF 模式等）
- **Directory**: 目录（URL、状态码、内容长度等）
- **HostPort**: 主机端口映射（主机、IP、端口）
- **Vulnerability**: 漏洞（类型、严重级别、CVSS 分数等）

## 错误处理

### 自动重试

| 错误类型 | 重试次数 | 等待时间 |
|----------|----------|----------|
| 5xx 服务器错误 | 3 次 | 1 秒 |
| 429 限流 | 3 次 | 5 秒 |
| 网络超时 | 3 次 | 1 秒 |
| 401 认证失败 | 1 次 | 自动刷新 token |

### 错误日志

所有错误详情记录在 `seed_errors.log` 文件中，包括：
- 时间戳
- 错误类型和状态码
- 请求数据（JSON）
- 响应数据（JSON）
- 重试次数

## 进度显示

```
🚀 Starting test data generation...
   Organizations: 15
   Targets: 225 (15 per org)
   Assets per target: 15

🏢 Creating organizations... [15/15] ✓ 15 created
🎯 Creating targets... [225/225] ✓ 225 created (domains: 157, IPs: 45, CIDRs: 23)
🔗 Linking targets to organizations... [225/225] ✓ 225 links created
🌐 Creating websites... [3375/3375] ✓ 3375 created
📝 Creating subdomains... [2355/2355] ✓ 2355 created (157 domain targets)
🔗 Creating endpoints... [3375/3375] ✓ 3375 created
📁 Creating directories... [3375/3375] ✓ 3375 created
🔌 Creating host port mappings... [3375/3375] ✓ 3375 created
🔓 Creating vulnerabilities... [3375/3375] ✓ 3375 created

✅ Test data generation completed!
   Total time: 45.2s
   Success: 12,000 records
   Errors: 0 records
```

## 常见问题

### Q: 为什么使用 API 而不是直接操作数据库？

A: 通过 API 可以：
- 测试完整的 API 流程（路由、中间件、验证、序列化）
- 发现 JSON 字段命名问题（camelCase vs snake_case）
- 模拟真实用户操作
- 验证业务逻辑和权限检查

### Q: 如何清空所有测试数据？

A: 使用 `--clear` 参数：
```bash
python seed_generator.py --clear
```

### Q: 生成数据时遇到 401 错误怎么办？

A: 检查用户名和密码是否正确：
```bash
python seed_generator.py --username admin --password admin
```

### Q: 如何生成更多数据？

A: 调整参数：
```bash
python seed_generator.py --orgs 100 --targets-per-org 50 --assets-per-target 20
```

### Q: 生成速度慢怎么办？

A: 可以增加批次大小（默认 100）：
```bash
python seed_generator.py --batch-size 200
```

## 技术细节

### JSON 字段命名

所有 API 请求使用 **camelCase** 字段名（符合前端规范）：

```python
# ✅ 正确
{"name": "example.com", "type": "domain"}

# ❌ 错误
{"name": "example.com", "type": "domain"}
```

### 独立 JSON 构造

不依赖 Go 后端的 DTO 结构体，使用 Python 字典独立构造 JSON：

```python
# ✅ 独立构造
data = {
    "name": "example.com",
    "type": "domain"
}

# ❌ 不要这样做
from go_backend.dto import CreateTargetRequest  # 不存在
```

这样可以发现 API 序列化问题，确保前后端字段命名一致。

## 开发和测试

### 运行单元测试

```bash
pytest
```

### 运行集成测试

```bash
# 1. 启动 Go 后端
cd ../../server
make run

# 2. 运行测试
cd ../tools/seed-api
pytest test_integration.py
```

## License

MIT
