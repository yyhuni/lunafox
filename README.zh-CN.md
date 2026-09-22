# LunaFox 部署

> **GENERATED / READ-ONLY PROJECTION**

[English](README.md)

## 安装

请安装支持 Linux 容器的 Docker，以及 2.24.0 或更高版本的 Docker Compose。

### 方式一：仓库安装

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
./install.sh
```

### 方式二：Release 压缩包

下载 `lunafox-<version>.zip`，然后运行：

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
./install.sh
```

### 方式三：直接使用 Docker Compose

在源码检出目录或解压后的目录中运行：

```console
docker compose up -d
```

## 登录与初始凭据

全新部署就绪后，打开配置的 LunaFox 地址，使用默认管理员账号登录：

| 用户名 | 密码 |
| --- | --- |
| `admin` | `admin` |

首次登录后，请前往账号设置及时修改密码。

如果忘记管理员密码，请在部署目录中运行：

```console
docker compose exec server resetadmin
```

该命令只显示一次新密码，并使管理员现有会话失效。

## Env 文件配置

首次启动前，请检查 `.env` 中的 `PUBLIC_HOST`、`PUBLIC_PORT`、`DATABASE_MODE` 和
`RELEASE_REGISTRY`。第一方镜像默认使用 Docker Hub；需要时显式选择 GHCR：

```dotenv
RELEASE_REGISTRY=docker.io
# RELEASE_REGISTRY=ghcr.io
```

首次成功启动后，`DATABASE_MODE`、`DB_PASSWORD` 和 `JWT_SECRET` 会持久化到
`lunafox_config`。之后在 `.env` 中将它们改成不同的非空值不会迁移或轮换正在运行的部署，而是快速失败；空的凭据输入会复用持久化值。
恢复和新建部署步骤请参阅[配置变更说明](docs/public-deployment.md#configuration-changes-after-first-start)。

## 日常操作命令

| 脚本 | 等价命令 | 行为 |
| --- | --- | --- |
| `./install.sh` | `docker compose up -d` | 首次启动和安全重跑；保留现有数据和 `.env`。 |
| `./start.sh` | `docker compose up -d` | 启动已有部署。 |
| `./restart.sh` | `docker compose up -d --force-recreate` | 使用当前配置重新创建容器。 |
| `./stop.sh` | `docker compose stop` | 停止所有常驻服务；保留全部数据。 |
| `./status.sh` | `docker compose ps` plus readiness checks | 只读摘要；只有完全就绪时才返回 0。 |
| `./logs.sh` | `docker compose logs --tail 200 --follow` | 只读日志；转发服务名称和 Compose 选项。 |
| `./uninstall.sh` | `docker compose down --remove-orphans` | 除非传入 `--purge --confirm`，否则保留 volumes 和 `.env`。 |
