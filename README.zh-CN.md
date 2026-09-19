# LunaFox 公共部署

> **GENERATED / READ-ONLY PROJECTION**

[English](README.md)

本仓库由
[`yyhuni/lunafox-private`](https://github.com/yyhuni/lunafox-private) 生成，包含经过审查的 Runtime 源码闭包，以及用于构建带版本号 Docker Compose 部署包的文件。变更应提交到私有源仓库，再通过受保护的发布 workflow 投影到这里。

## 安装

请安装支持 Linux 容器的 Docker，以及 2.24.0 或更高版本的 Docker Compose。公共 `main` 分支是完整的部署快照，因此检出源码后即可直接启动：

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
./install.sh
```

`./install.sh` 会读取 `.env`，执行一次 `docker compose up -d`，并等待部署真正就绪。同一份检出内容也可以不使用脚本，这也是原生 Windows PowerShell 路径：

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
docker compose up -d
```

每个新的 GitHub Release 也会包含一份等价的部署归档：

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
./install.sh
```

两种路径都包含 `.env`、`.env.example`、`compose.yaml`、发布 manifest 和 Engine inventory。启动前请检查 `.env` 中的 `PUBLIC_HOST`、`PUBLIC_PORT` 和 `DATABASE_MODE`。第一方镜像默认使用 Docker Hub：

```dotenv
RELEASE_REGISTRY=docker.io
```

如需让完整的第一方 Runtime 和 Engine 闭包使用 GHCR，请在首次启动前修改为：

```dotenv
RELEASE_REGISTRY=ghcr.io
```

只接受 `docker.io` 和 `ghcr.io`。LunaFox 不会自动回退或混用 Registry。Compose 会在 `docker compose up -d` 过程中校验配置并拉取缺失镜像。以下命令仅用于诊断或预拉取：

```console
docker compose config --quiet
docker compose pull
docker compose ps
```

`PUBLIC_URL` 由内部派生。`DB_USER=postgres` 和 `DB_NAME=lunafox` 是可编辑的默认值。嵌入模式会生成空的 `DB_PASSWORD` 和 `JWT_SECRET` 值一次，并将其保存在 `lunafox_config` volume 中。对于已有的远程 PostgreSQL 数据库，请在首次启动前设置 `DATABASE_MODE=external`、`DB_HOST`、`DB_SSLMODE` 以及真实的 `DB_PASSWORD`；LunaFox 会应用自身 schema，但不会创建远程数据库或迁移已有数据。

Windows 用户解压 Release ZIP 后，在原生 PowerShell 中执行相同的 Compose 命令。PowerShell 下载器、WSL 终端、Docker context 名称检查和 Loki Docker logging plugin 都不属于此部署路径。

日常操作可以使用生命周期脚本，也可以使用等价的 Docker Compose 命令：

| 脚本 | 等价命令 | 行为 |
| --- | --- | --- |
| `./install.sh` | `docker compose up -d` | 首次启动和安全重跑；保留现有数据和 `.env`。 |
| `./start.sh` | `docker compose up -d` | 启动已有部署。 |
| `./restart.sh` | `docker compose up -d --force-recreate` | 使用当前配置重新创建容器。 |
| `./stop.sh` | `docker compose stop` | 停止所有常驻服务；保留全部数据。 |
| `./status.sh` | `docker compose ps` plus readiness checks | 只读摘要；只有完全就绪时才返回 0。 |
| `./logs.sh` | `docker compose logs --tail 200 --follow` | 只读日志；转发服务名称和 Compose 选项。 |
| `./uninstall.sh` | `docker compose down --remove-orphans` | 除非传入 `--purge --confirm`，否则保留 volumes 和 `.env`。 |

只有在所有服务、常驻 Agent 和公共 endpoint 都就绪后，`install.sh`、`start.sh` 和 `restart.sh` 才会打印 `SUCCESS`、公共 HTTPS 地址以及 `admin / admin` 首次登录提示。否则会打印 `FAILED`，指出失败服务并保留部署以便检查。脚本需要 Bash 3.2 或更高版本，并可从任意工作目录运行。

Docker Compose 仍然提供完整的生命周期接口：

```console
docker compose stop
docker compose start
docker compose restart
docker compose down
```

`down` 会保留命名 volumes，包括生成的配置 secrets。删除数据是单独且明确的运维操作。LunaFox 不会自动迁移或清除旧的脚本管理部署；采用新包前请先检查并备份旧状态。

初始化、恢复、Registry、证书和安全边界请参阅 [`docs/public-deployment.zh-CN.md`](docs/public-deployment.zh-CN.md)。

## 发布身份

发布 workflow 会把 Server、Frontend、Nginx、Agent、Bootstrap 和 Engine 引用解析为跨 Registry 不可变的 digest。只有最终 manifest 通过验证后才会构建统一部署包，并发布其 SHA-256 记录；如果同名发布资产已经存在且内容冲突，workflow 会拒绝替换。

私有发布运行会在请求受保护的公共导出合并后结束。私有运行显示绿色只代表已请求公共交接；最终发布状态以公共仓库 workflow 和 GitHub Release 为准。

## 源码与安全范围

导出的 Runtime 源码闭包包含 Frontend、Server、Contracts、`engine-go`、Proto、第一方 Extensions、Nginx、Bootstrap，以及经过审查的测试和构建元数据。Agent 源码和私有签名材料不在公共源码边界内。封闭 Agent artifact 的条款见 [`NOTICE-CLOSED-ARTIFACTS.md`](NOTICE-CLOSED-ARTIFACTS.md)。

当前 Agent credential 仍是长期有效的八字符十六进制 bearer。本次发布没有加入 Agent TLS certificate-chain identity、replay protection、rotation 或 revocation。TLS 和 credential hardening 需要单独的安全变更。本部署用于由操作者拥有或控制的基础设施上的 self-hosted 场景。

导出的第一方 Runtime 源码和部署文件采用 SPDX `GPL-3.0-only` 许可。

Engine Runtime Images 和 Engine Packages 是公共 artifact。第一方 Engine 源码采用 GPL-3.0-only。随附的第三方组件继续受其适用的许可证和归属条款约束。
