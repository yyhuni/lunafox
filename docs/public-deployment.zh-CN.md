# 公共 Docker Compose 部署

> **GENERATED / READ-ONLY** - 请修改私有源，并通过受保护的发布 workflow 重新生成此投影。

[English](public-deployment.md)

## 安装来源

公共 `main` 分支和每个 Release 中唯一的 `lunafox-<version>.zip` 都包含等价的部署快照。每个快照包括 `compose.yaml`、`.env`、`.env.example`、经过验证的最终 release manifest、双 Registry Engine inventory、Loki 和 Alloy 配置、默认 fingerprint corpus、所需 wordlists，以及带共享 helper 的 Bash 生命周期脚本。产品和 Engine 镜像带有 digest；PostgreSQL、Redis、Loki 和 Alloy 则由多平台 manifest digest 固定。

请保留解压后的目录：`.env` 是由主机负责的安装配置，相对资源路径都从此目录解析。检查 `PUBLIC_HOST` 和 `PUBLIC_PORT`；内部 HTTPS `PUBLIC_URL` 由它们派生。写入 `DB_PASSWORD` 或 `JWT_SECRET` 后，请用 `chmod 600 .env` 限制文件权限；生命周期脚本在进行任何变更前也会强制使用相同模式。

### 从公共仓库安装

在 Bash 环境中检查 `.env` 并运行生命周期脚本。它会执行一次 `docker compose up -d`，并等待部署完全就绪：

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
./install.sh
```

同样支持 `docker compose up -d`，这是原生 Windows PowerShell 入口：

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
docker compose up -d
```

### 从 Release 安装

从对应的 GitHub Release 下载 `lunafox-<version>.zip` 及其 SHA-256 元数据，按本地策略需要时完成校验，然后解压到永久目录：

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
./install.sh
```

解压后的 ZIP 也支持直接使用 Compose：

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
docker compose up -d
```

Windows 操作者解压同一份 ZIP 后，在原生 PowerShell 中运行 Compose 命令。

### 选择 Registry

两种安装来源在 `.env` 中都默认使用 Docker Hub：

```dotenv
RELEASE_REGISTRY=docker.io
```

如需使用 GHCR，请在首次启动前修改为：

```dotenv
RELEASE_REGISTRY=ghcr.io
```

只接受 `docker.io` 和 `ghcr.io`。所选值适用于完整的第一方 Runtime、Agent、Engine、bootstrap 和 upgrader 闭包。LunaFox 不会回退到另一个 Registry，也不会在一次部署中混用两个 Registry。成功完成原地升级后，持久化的 override 会绑定到原 Registry；更换 Registry 需要新部署，或需要保留应用数据和配置的明确迁移。

`DATABASE_MODE=embedded` 是默认值，并启用捆绑的 PostgreSQL 服务。`DB_PORT=5432`、`DB_USER=postgres` 和 `DB_NAME=lunafox` 是可编辑的默认值。此模式下保持 `DB_HOST` 和 `DB_SSLMODE` 为空。`DB_PASSWORD` 为空时，会在首次初始化时生成随机值。

如需使用 Compose 项目之外已经存在的数据库，请在首次启动前设置以下全部配置：

```dotenv
DATABASE_MODE=external
DB_HOST=database.example
DB_PORT=5432
DB_USER=postgres
DB_NAME=lunafox
DB_SSLMODE=require
DB_PASSWORD='the existing remote database password'
```

保持 `COMPOSE_PROFILES=${DATABASE_MODE:-embedded}` 不变，使所选模式控制服务图。External 模式要求 PostgreSQL 服务器可访问、账户和数据库已存在，并且拥有应用 LunaFox schema migrations 的权限。LunaFox 不会创建这些资源，也不会从捆绑数据库复制数据。`DB_SSLMODE` 接受 `disable`、`allow`、`prefer`、`require`、`verify-ca` 或 `verify-full`；`verify-ca` 和 `verify-full` 使用容器的系统信任根。本版本不支持挂载自定义 CA 文件。

`JWT_SECRET` 仍然是可选项，为空时会生成。配置输入只会作为 Compose secret files 传给 initializer，不得提交到仓库或打印到支持日志中。

## 启动与生命周期

主机前置条件是支持 Linux 容器的 Docker，以及 2.24.0 或更高版本的 Docker Compose。从解压目录运行：

```console
docker compose up -d
docker compose ps
```

Compose 会在 `docker compose up -d` 期间校验配置并拉取缺失镜像。`docker compose config --quiet` 是可选的配置诊断，`docker compose pull` 是可选的预拉取步骤；安装不要求执行它们。

在 Windows 上，直接在 PowerShell 中运行这些命令。在 macOS、Linux、Docker Desktop 或 OrbStack 上，使用 Docker CLI 已选择的 Docker endpoint。LunaFox 不会检查或 allowlist 主机 OS、发行版、Docker 品牌或 context 名称。连接、平台、绑定路径和发布端口错误由 Docker 自身报告。

普通生命周期如下：

```console
docker compose stop
docker compose start
docker compose restart
docker compose down
```

## 管理员登录与密码找回

全新部署执行 `docker compose up -d` 并报告就绪后，打开 `PUBLIC_URL`，使用默认管理员账号登录：

| 用户名 | 密码 |
| --- | --- |
| `admin` | `admin` |

首次登录后，请前往账号设置及时修改密码。

如果忘记管理员密码，请在部署目录中运行：

```console
docker compose exec server resetadmin
```

该命令只重置现有的 `admin` 账号，只显示一次新密码，并使管理员现有会话失效。账号不存在时不会创建账号。

### 生命周期脚本

每个快照还会在部署根目录提供七个小型 Bash 入口，以及共享 helper `lunafox-lifecycle.sh`。它们封装相同的 Compose 命令，补充 Docker 无法表达的检查，并且不会维护第二套部署状态：

| 脚本 | 等价 Compose 命令 | 行为 |
| --- | --- | --- |
| `./install.sh` | `docker compose up -d` | 首次启动和安全重跑。保留已有 `.env` 及全部命名 volume。 |
| `./start.sh` | `docker compose up -d` | 启动已有部署。 |
| `./restart.sh` | `docker compose up -d --force-recreate` | 将当前 `.env` 和 `compose.override.yaml` 应用到每个容器。 |
| `./stop.sh` | `docker compose stop` | 停止所有常驻服务并保留全部数据。 |
| `./status.sh` | `docker compose ps` plus the readiness checks | 只读摘要；只有完全就绪时才返回 0。 |
| `./logs.sh` | `docker compose logs --tail 200 --follow` | 只读日志访问，并转发服务名称和 Compose 选项。 |
| `./uninstall.sh` | `docker compose down --remove-orphans` | 移除容器和项目网络；默认保留 volumes、`.env` 和 `compose.override.yaml`。 |

只有在 one-shot tasks、核心服务健康状态、辅助服务稳定性、常驻 Agent 的 claim-ready state 和公共 HTTPS endpoint 全部通过后，`install.sh`、`start.sh` 和 `restart.sh` 才报告成功。`install.sh` 允许十五分钟，另外两个允许五分钟；`LUNAFOX_READY_TIMEOUT_SECONDS` 可以为单次调用覆盖此窗口。

失败时会打印 `FAILED`，指出失败的服务或检查，并保留部署现场：容器、volumes 和配置保持 Compose 创建后的状态，不会回滚。诊断输出顺序固定为结论、阶段、保留现场、后续命令；如果已经识别服务，会指向 `./logs.sh <service>`，否则指向 `./logs.sh`。

`install.sh` 会描述它识别到的运行类型，而不是始终叙述为首次启动：没有任何 LunaFox 数据的目录属于首次启动；只有 volumes 而没有容器的目录会在保留命名 volumes 和已安装 Engine 的情况下重建容器，不属于升级；运行中的部署会在保留配置和 volumes 的情况下启动。当这种重建部署在 `bootstrap` 等首次启动任务上失败时，footer 会在 `logs.sh` 和 `status.sh` 提示之后增加一行可选信息：如果可以丢弃保留数据，只有 `./uninstall.sh --purge --confirm` 能删除命名 volumes 并重新开始。

`install.sh` 只接受 `--help`，从 `.env` 读取全部设置，并且恰好运行一次 `docker compose up -d`。它不会拉取、构建、删除容器或删除数据。默认的 `uninstall.sh` 是可恢复的：它保留命名 volumes、`.env`、发布目录，以及已完成 Upgrade Operation 写入的常规 `compose.override.yaml`。`./uninstall.sh --purge --confirm` 会先验证并删除当前 `compose.yaml` 声明的 LunaFox 命名 volumes，再移除该版本覆盖层，以便在同一目录中进行干净重装；`.env` 和发布目录仍会保留。

脚本需要 Bash 3.2 或更高版本，并可从任意工作目录运行。Windows 继续在 PowerShell 中使用直接 Compose 命令。

### 部署锁与恢复

生命周期变更和 Upgrade Operation 绝不会重叠。两者都会获取部署级锁 `.lunafox-lifecycle.lock`，而 `./status.sh` 会不等待地报告该锁：

```console
./status.sh
LunaFox:   address: https://localhost:443
LunaFox:   lock: upgrade recovery fence (operation <id>)
...
LunaFox deployment status: starting
```

锁不会自动回收，因为不能把一个缓慢的操作误判为已死亡。确认没有生命周期命令和升级正在运行后，再手工删除目录：

```console
cat .lunafox-lifecycle.lock/owner
rm -rf .lunafox-lifecycle.lock
```

直接执行的 `docker compose` 命令无法取得此锁，因此不受互斥保护。中断升级时，`lunafox_upgrade_state` 中的 upgrade journal 仍是事实来源。

内部 Agent 和受限 upgrader 都是 Compose 服务，因此四个命令都会包含它们。`down` 会移除容器和项目网络，同时保留命名 volumes，包括 `lunafox_upgrade_state`。除非明确要删除并已备份永久状态，否则不要添加 `--volumes`。本部署中的任何命令都不会自动迁移、重置或删除旧部署。

## 初始化

Compose 通过 one-shot services 表达初始化：

1. `config-init` 校验 database mode、派生的 `COMPOSE_PROFILES` 值和连接输入，生成或采用允许的 secrets，并将 database mode、database password 和 JWT secret 以 mode-0600 文件持久化到 `lunafox_config`。
2. `agent-preflight` 使用已发布的 Agent image，通过真实的 sibling container round trip 校验 Docker API、精确的 named-volume identity、volume-subpath isolation 和只读 execution mounts。
3. `cert-init` 在 `lunafox_ssl` 中创建或校验证书和私钥。
4. 在 embedded 模式中，PostgreSQL 会在 configuration initialization 后启动，`migrate` 等待其 health；在 external 模式中，捆绑的 PostgreSQL 服务不存在，`migrate` 直接连接配置的 host。
5. `bootstrap` 等待 preflight 和 migration 都完成，安装经过验证的 Engine inventory 与默认资源，然后创建或校验内部 Agent credential。
6. Server 只在 bootstrap 成功后启动。Agent 在 bootstrap 和 Server health 都成功后启动；Nginx 在 certificate、Server 和 Frontend readiness 之后启动。

重复启动会复用 `lunafox_config` 中的文件。首次初始化成功后，改变 `DATABASE_MODE` 会在任何数据库 consumer 启动前失败。之后提供的非空 `DB_PASSWORD` 或 `JWT_SECRET` 如果与持久化值不同也会失败；Compose 不会轮换在线 credential。没有 mode record 的旧 configuration volume 默认采用 embedded mode。从此类 volume 采用 external mode 时，需要提供与持久化值匹配的明确 `DB_PASSWORD`。

恢复路径以及采用其他部署配置的支持方式，请参阅[首次启动后的配置变更](#configuration-changes-after-first-start)。

对于 embedded mode，请将 `lunafox_config` 与 `lunafox_postgres` 一起备份。对于 external mode，请按照 operator 批准的流程，将 `lunafox_config` 与外部数据库一起备份。只删除 configuration volume 会丢失连接所选数据库所需的 password。

credential 是 `lunafox_agent_state` 中 mode-0600 的普通 JSON 文件，绝不会通过主机环境或 Docker container metadata 传递。Registration、database binding 和 credential publication 是串行的。重复成功执行 `docker compose up -d` 会复用同一个已注册 Agent。缺失、格式错误、权限过宽、发布不完整、被删除或与数据库不一致的 identity state 都会返回明确的修复错误，绝不会静默注册另一个 Agent。

默认资源也遵循相同的 fail-closed 规则。全新状态会一次性导入打包的 fingerprint corpus 和 wordlists。之后的 bootstrap 运行会校验原始 records、files、hashes、sizes、descriptions 和 tags，同时保留额外的用户资源。默认值缺失、部分删除或发生变化时会返回明确的修复错误；bootstrap 绝不会填补或覆盖这些状态。已经初始化但被完全清空的 corpus 不会被静默重新 seed。

preflight 会在业务 bootstrap 之前检查实际 Docker socket、Linux execution node、named-volume subpaths 和只读 execution mounts。常驻 Agent 启动时会再次检查 socket、平台、API version 和精确 volume identities。不支持的能力会明确失败；不存在更弱的 whole-volume fallback。Server 永远不会收到 Docker socket。

Compose 管理的 Agent 禁用了容器内 self-update。受限 upgrader 会根据 release Manifest 更新固定的 `agent` service。现有远程 Agent 安装和 `update_required` self-update 行为不变。

## 首次启动后的配置变更

首次成功运行 `config-init` 后，配置边界就会建立。它会将 `DATABASE_MODE`、`DB_PASSWORD` 和 `JWT_SECRET` 持久化到 `lunafox_config`；`COMPOSE_PROFILES` 必须继续由 `DATABASE_MODE` 派生。之后提供的非空输入都会与持久化状态进行校验。

本版本不支持以下任何一种原地操作：

- 不支持在线数据库模式迁移：通过修改 `.env` 在 `embedded` 和 `external` 之间切换 `DATABASE_MODE`，不属于原地操作。
- 不支持在线凭据轮换：通过修改 `.env` 轮换在线数据库密码或 `JWT_SECRET`，不属于原地操作。
- 在 `lunafox_postgres` 与外部 PostgreSQL 服务器之间自动复制数据。
- 删除或替换 `lunafox_config`，以强制选择新配置。

简而言之，本版本不支持在线数据库模式迁移，也不支持在线凭据轮换。

如果已有部署因误改配置而被阻塞：

1. 在 `.env` 中恢复原来的 `DATABASE_MODE` 和 `COMPOSE_PROFILES`。
2. 恢复原来的非空 `DB_PASSWORD` 和 `JWT_SECRET`，或者删除新填入的值，使空输入复用持久化文件。
3. 保留 `lunafox_config`；embedded mode 还要保留 `lunafox_postgres`。external mode 则要保留外部数据库及其由运维管理的备份。不要执行 `docker compose down --volumes`。
4. 运行 `docker compose up -d`；如果初始化仍然失败，在修改任何持久化状态前先检查 `docker compose logs config-init`。

如需采用其他模式或凭据，请使用独立的新部署：

1. 保留旧部署，并对 embedded mode 的 `lunafox_config` 和 `lunafox_postgres`，或 external mode 的外部数据库，分别执行已验证的备份。
2. 创建新的部署目录，在第一次 `docker compose up -d` 前写入最终的 `.env`、`DATABASE_MODE`、`COMPOSE_PROFILES`、连接参数和 secrets。
3. 如果必须保留数据，请使用运维方已验证的 PostgreSQL backup/restore 或 export/import 流程。在切换流量前验证 schema、应用数据、连接访问和 health。LunaFox 不会复制数据、修改远程 role，也不提供自动 rollback。
4. 新部署验收完成前，保留旧部署作为 rollback 目标。

凭据轮换仍需在此 Compose workflow 之外协调完成。PostgreSQL 密码变更必须与数据库 role 和新部署的 `DB_PASSWORD` 同步；变更 `JWT_SECRET` 可能使现有 session 失效。不要直接编辑 `lunafox_config` 中的文件。

## 系统更新

发布包会在 `compose.yaml` 中固定其 `stable` 或 `canary` channel 和 public metadata source。没有主机 update script：版本变化通过 Upgrade Operation 和受限 Compose upgrader 完成，生命周期脚本既不复制也不绕过该状态机。所选 Registry 来自首次启动前的 `.env`，之后由升级保留。Server 通过 HTTPS 获取当前 schema-v3 channel record，校验其受限的 Manifest path 和原始 SHA-256，然后按 digest 将不可变 Manifest 缓存到 `lunafox_upgrade_state`。同时，它会从按版本隔离的 `manifests/<release-tag>/runtime-composition.json` 路径下载 manifest 绑定的 composition，并将已验证字节保存到 `.lunafox/upgrade/compositions/<composition-core-digest>.json`。运行中的 Server binary 仍是 `upgrade.compatibilityRange` 的输入；host 的 confirmed deployment inventory 才是 candidate 可用性和组件比较的 baseline，因此已确认的 frontend-only release 不会再次被提供。

管理员可以通过现有 frontend update control 检查、确认并启动符合条件的更新。在 Server 暂停调度或取消 work 之前，支持 v2 的 host 会返回一个只读 scope plan，将 candidate manifest、composition、confirmed baseline 和 live container digest 绑定在一起。只有确认过 dynamic frontend upstream 能力、且差异严格为 `{frontend}` 时，才会产生 `frontend_only`；它只在无 dependencies、无 build 的情况下 pull 和 recreate `frontend`。deployment lock 获取后会重新验证该 plan；stale plan 会在副作用前失败，绝不会自行扩大为 `full`。Server 只通过共享 Unix Socket 发送 Operation ID、固定 action 和 Manifest digest。即使浏览器关闭或 Server 被重建，Compose 管理的 upgrader 仍会继续。它没有 network port，并使用固定的 project、file set、command argv、service allowlist 和 `--no-deps` recreation。Server 永远不会收到 Docker Socket。

upgrader 的可写 Docker Socket 赋予它控制本地 Docker daemon 的能力。该服务还挂载解压后的 package directory，以便读取 `compose.yaml` 和 `.env`。full operation 会在完成 service、migration、Agent、health 和 digest 校验后原子安装 `compose.override.yaml`；`frontend_only` operation 只暂存修改 frontend image 的 patch，只有 Server 发送 `confirm` 后才晋级持久化 override。请将该 override 与解压目录一起保留：普通的 `docker compose up -d`、`start` 和 `restart` 会自动保留已确认的 image 和 version target，不会重写 database、public address 或 secret 设置。

host 只有在完整验证后才会替换 confirmed deployment state。对于 `frontend_only`，只有 frontend digest、health 和 public Nginx response 收敛后才会记录新的 component inventory；失败或部分执行绝不会被猜测为新的 baseline。

完成 receipt 只证明受限的 Compose 部署工作结束。Server 会将它与 database Operation、journal、migration、service、API 和 Agent evidence 对账后才报告成功。migration 失败或结果不确定时需要恢复；Agent 超时需要人工处理。当前 `release-candidate` policy 使用冻结的 `000001` migration baseline，仍不提供保留数据的 rollback 或自动备份。生产恢复只能使用已验证的备份恢复或批准的前向修复；`down` migration 仅用于测试 teardown。

自动更新支持兼容的 image-only releases。明确只支持 `schema-v1` 的 host 沿用既有 `full` 路径；缺少 composition cache 也只允许 `full` planning。损坏或篡改的 composition、无效的 v2 capability/plan 数据以及 stale scoped plan 会在 scheduler、cancellation 或 Compose 副作用前 fail closed，不会静默降级。声明 migration 或修改任何非 frontend 组件的 candidate 同样保持 `full` 路径。改变 Compose 结构、named volumes、initialization resources 或 upgrader protocol 的 release 必须声明不兼容；请下载新的 deployment ZIP，保留文档说明的状态，检查其 README，并从新的解压目录运行 Compose。运行中的 release 必须满足 Manifest 的 `upgrade.compatibilityRange`；否则更新仍可见，但不能创建自动 Upgrade Operation。如果主机无法向 upgrader container 暴露本地 Docker Socket，这也是手工 fallback。

## 日志

所有常驻服务都使用 Docker 的 `json-file` driver，设置 `max-size=10m` 和 `max-file=3`。`./logs.sh` 会以相同的有界输出跟随所有服务；传入服务名称或 Compose logs 选项可以缩小范围，例如 `./logs.sh server`。普通 Alloy 容器只通过 Docker socket 读取带有 LunaFox Server 和内部 Agent 标签的输出，并将其发送到本地 Loki 服务。Alloy 将读取位置保存到 `lunafox_alloy`，因此 collector 重启不会重新播放全部保留日志。

collector 保留现有 selectors `{component="server",container_name="lunafox-server"}` 和 `{agent_id="<id>",container_name="lunafox-agent"}`。它不会采集自己的输出。不安装也不需要 Loki Docker plugin。Docker socket 访问仅限于受限 upgrader、Compose Agent、其 one-shot capability preflight、collector 和仅开发使用的工具；常驻 Server 没有 Docker control path。

## 证书与恢复

`cert-init` 会创建带有 `PUBLIC_HOST` DNS 或 IP SAN 的自签名 RSA 证书。已有证书文件必须是权限受限的普通文件、未过期、与配置的 host 匹配，并且与私钥成对。部分或无效状态会失败，而不是被覆盖。本版本不提供 ACME、用户证书导入或自动续期。

使用 `docker compose logs config-init agent-preflight migrate bootstrap cert-init agent upgrader` 诊断失败的 one-shot 或 Agent 启动。修正配置或明确修复命名 volume 状态，然后再次执行 `docker compose up -d`。Nginx readiness probe 使用 `127.0.0.1` 以匹配其 IPv4 listener，即使在 `localhost` 优先解析为 IPv6 的主机上也是如此。对于普通初始化，Compose exit status、dependency conditions、container state 和 service health 就是部署状态。Upgrade journal 和完成 receipt 文件只存在于 `lunafox_upgrade_state`，并由 Upgrade Operation API 对账。

从只把 credential 存在 `.env` 的包升级时，新包首次启动仍需保留已有的非空 `DB_PASSWORD` 和 `JWT_SECRET`。`config-init` 会将它们复制到 `lunafox_config`。完成迁移后，空输入会复用持久化文件。冲突值会明确失败；本版本不支持在线凭据轮换。在 embedded 和 external PostgreSQL 之间迁移现有安装不属于原地操作；请参阅[首次启动后的配置变更](#configuration-changes-after-first-start)，了解独立部署和运维数据迁移路径。

## 发布与安全边界

这是单节点 Compose release-candidate 部署，不提供滚动升级、自动备份、保留数据 rollback 或跨节点恢复。已发布的 `000001` migration baseline 不可变，也不承诺与已有持久化部署兼容。Release 生成和契约检查使用隔离 fixture，不会触碰操作者的 volumes。

当前 Agent authentication credential 仍是长期有效的八字符十六进制 bearer。Agent TLS certificate-chain identity、replay protection、rotation 和 revocation 仍延期。TLS 和 credential hardening 需要单独的安全变更。本 package 用于由操作者拥有或控制的基础设施上的 self-hosted 场景；封闭 Agent artifact 仍受 `NOTICE-CLOSED-ARTIFACTS.md` 约束。

导出的第一方 Runtime 源码和部署文件采用 SPDX
`GPL-3.0-only` 许可。

Engine Runtime Images 和 Engine Packages 是公共 artifact。第一方 Engine 源码采用 GPL-3.0-only。随附的第三方组件继续受其适用的许可证和归属条款约束。
