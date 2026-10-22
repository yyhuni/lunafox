<h1 align="center">LunaFox · 月狐</h1>

<p align="center">
  <b>自托管攻击面管理平台（ASM）| 面向授权安全测试与防御研究的资产发现与安全自动化系统</b>
</p>

<p align="center">
  <a href="https://github.com/yyhuni/lunafox/stargazers"><img src="https://img.shields.io/github/stars/yyhuni/lunafox?style=flat-square&logo=github" alt="GitHub stars"></a>
  <a href="https://github.com/yyhuni/lunafox/network/members"><img src="https://img.shields.io/github/forks/yyhuni/lunafox?style=flat-square&logo=github" alt="GitHub forks"></a>
  <a href="https://github.com/yyhuni/lunafox/issues"><img src="https://img.shields.io/github/issues/yyhuni/lunafox?style=flat-square&logo=github" alt="GitHub issues"></a>
  <a href="https://github.com/yyhuni/lunafox/blob/main/LICENSE"><img src="https://img.shields.io/github/license/yyhuni/lunafox?style=flat-square" alt="License"></a>
</p>

<p align="center">
  <a href="#功能特性">功能特性</a> •
  <a href="#快速开始">快速开始</a> •
  <a href="#文档">文档</a> •
  <a href="#反馈与贡献">反馈与贡献</a>
</p>

<p align="center">
  <sub>关键词：ASM | 攻击面管理 | 资产发现 | 授权安全测试 | 分布式扫描 | 全局搜索 | 漏洞管理</sub>
</p>

**简体中文** | [English](README.en.md)

---
## 在线 Demo

[公开演示站](https://lunafox-demo.vercel.app/) 使用 mock 数据展示界面，无需登录，也未接入真实后端。

<p align="center">
  <img src="docs/images/zh/overview.png" alt="总览" width="24%">
  <img src="docs/images/zh/workflows.png" alt="扫描工作流" width="24%">
  <img src="docs/images/zh/scan-history.png" alt="扫描历史" width="24%">
  <img src="docs/images/zh/nodes.png" alt="Agent 节点" width="24%">
</p>

---
## 文档

- [公共部署指南](docs/public-deployment.md) - 安装、日常运维、数据库、升级与恢复
- [Release Notes](release-notes/README.md) - 每个版本的变更记录
- [贡献指南](CONTRIBUTING.md) - Issue、Pull Request 与贡献约定

---
## 功能特性

### 内置引擎

以下为默认内置引擎；引擎以 Engine Package 形式分发，可按需扩展。

| 引擎 | 工具 | 说明 |
|------|------|------|
| 子域名扫描 | Subfinder, PureDNS, MassDNS | 被动收集 + 主动枚举 |
| 端口扫描 | Naabu | 自定义端口范围 |
| 站点发现 | HTTPX | 探测状态码、标题、技术栈 |
| 指纹识别 | Finger | 多源指纹库 |
| URL 收集 | Katana, Waymore, HTTPX | 历史数据 + 主动爬取 |
| 目录扫描 | FFUF | 智能校准、递归与并发控制 |
| 安全检测 | Nuclei | 模板化漏洞检测 |
| 站点截图 | Chromium, HTTPX | 截图与页面信息采集 |

### 平台能力

| 功能 | 说明 |
|------|------|
| 目标管理 | 域名 / IP / CIDR 目标 |
| 组织管理 | 多组织与资产归属 |
| 全局搜索 | 表达式语法，多字段组合查询 |
| 漏洞管理 | 审核、复测与资产关联 |
| 定时扫描 | Cron 周期任务 |
| 扫描编排 | 工作流组合、冻结与历史 |
| 分布式执行 | 多 Agent 节点与负载调度 |
| 字典与指纹 | 词表、指纹库与 Nuclei POC 管理 |
| 通知推送 | Discord 与 Webhook |
| MCP 集成 | AI 客户端接入调查与受控操作 |
| 系统升级 | 应用内升级、进度与恢复 |

### 全局资产搜索

- 多类型搜索：Website 与 Endpoint 资产
- 表达式语法：`=` 包含、`==` 精确、`!=` 不等于
- 逻辑组合：`&&`（AND）与 `||`（OR）
- 多字段查询：url、host、title、statusCode、tech、responseHeaders、hasScreenshot
- CSV 导出：流式导出全部搜索结果

```bash
# basic search
host="api"
statusCode==200
tech="nginx"

# combinations
host="api" && statusCode==200
tech="vue" || tech="react"
```

### 可视化界面

- 数据统计：资产、扫描、漏洞和 Agent 集群状态
- 实时进度：扫描进度、Engine 执行状态和历史记录
- 通知推送：Discord 与 Webhook
- 界面语言：简体中文与 English 可切换

---
## 快速开始

### 环境要求

- 支持 Linux 容器的 Docker Engine，以及 Docker Compose 2.24.0 或更高版本
- 生命周期脚本需要 Bash 3.2 或更高版本；Windows 请直接在 PowerShell 运行 Compose
- 为 `PUBLIC_PORT` 准备可用端口；默认 443
- 能访问 Docker Hub、GHCR 或已配置的镜像仓库
- 为镜像、数据库数据、扫描结果和日志预留磁盘空间

### 安装

公共仓库和每个 `lunafox-<version>.zip` Release 都包含等价的部署快照。

**方式 1：从公共仓库安装**

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
./install.sh
```

**方式 2：从 Release 压缩包安装**

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
./install.sh
```

**方式 3：直接使用 Docker Compose**

```console
docker compose up -d
```

方式 1、2 会运行安装脚本，完成参数校验、镜像预热和就绪检查；方式 3 会跳过这些步骤，仅建议在自行处理时使用。

从其他机器访问时，设置公网地址和端口：

```console
./install.sh --public-host luna.example.com --public-port 8443
```

首次启动会为 `PUBLIC_HOST` 生成自签名证书；本版本不自动管理证书。数据库设置见[数据库环境变量](docs/public-deployment.zh-CN.md#数据库环境变量)，首次启动后的配置变更见[配置变更](docs/public-deployment.zh-CN.md#首次启动后的配置变更)。

---
## 反馈与贡献

- 发现 Bug 或改进建议：提交 [Issue](https://github.com/yyhuni/lunafox/issues)
- 想参与开发：请先阅读 [贡献指南](CONTRIBUTING.md)
- 安全问题：请勿公开披露未修复漏洞，按贡献指南私下联系维护者

---
## 安全与合规说明

1. 本项目面向授权安全测试、防御研究与自有资产攻击面管理
2. 使用者必须确保已获得目标系统、数据和环境的明确合法授权
3. 默认 TLS 证书为自签名；本版本不自动管理证书
4. Agent 使用长期有效的 bearer 令牌；TLS 身份、重放保护、轮换与吊销尚未实现
5. 单机部署不提供自动备份、保留数据的回滚或跨节点恢复

请勿将本项目用于任何未经授权的扫描、入侵或破坏行为。

---
## Star History

[![Star History Chart](https://star-history.dera.page/svg?repos=yyhuni/lunafox&type=Date)](https://star-history.dera.page/#yyhuni/lunafox&Date)

---
## 许可证

部署文件与第一方公开的 Runtime/Engine 源码采用 [GPL-3.0-only](LICENSE)；Engine Runtime Images 与 Engine Packages 是公开产物；第三方组件遵循各自的许可证与署名条款。

> **GENERATED / READ-ONLY PROJECTION（生成投影，只读）。** 由 LunaFox 私有源码生成；发现问题请带上对应 release tag 按 [CONTRIBUTING.md](CONTRIBUTING.md) 反馈。
