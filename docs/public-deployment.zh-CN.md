# 公共 Docker Compose 部署

> **GENERATED / READ-ONLY** - 请修改私有源，并通过受保护的发布 workflow 重新生成此投影。

[English](public-deployment.md)

本文是单节点、自托管 LunaFox 部署的运维手册。一套 Docker Compose 栈运行控制面、前端、PostgreSQL 或外部数据库、Loki 与 Alloy 日志，以及负责启动 Engine 容器的常驻 Agent。

## 部署模型

LunaFox 将决策与执行分离：

- Server 规划每个任务，将其冻结为不可变执行计划，并按需流式产生输入。
- 每个 worker 主机上的 Agent 领取兼容计划，把输入物化为只读文件，并为每个任务启动一个 Engine 容器。
- Engine 容器运行镜像内工具并上报类型化结果；它们从不查询数据库。
- 常驻 Server 没有 Docker 控制通道。Docker socket 访问仅限于受限 upgrader、Compose Agent 及其能力预检，以及日志采集器。

产品与 Engine 镜像都带 digest。PostgreSQL、Redis、Loki 和 Alloy 由多平台 manifest digest 固定。部署目录是所有权单位：`.env` 归主机所有，相对资源路径从它解析，完成升级后写入的 `compose.override.yaml` 必须与它一起保留。

## 环境要求

- 支持 Linux 容器的 Docker Engine。
- Docker Compose 2.24.0 或更高版本。
- 生命周期脚本需要 Bash 3.2 或更高版本。Windows 上请直接在 PowerShell 中运行 Compose。
- 为 `PUBLIC_PORT` 准备一个可用主机端口；默认是 443。
- 能访问 Docker Hub、GHCR，或 `.env` 中配置的自定义 Registry。
- 为镜像、数据库数据、扫描结果和保留日志预留磁盘空间。

## 部署包包含什么

公共 `main` 分支和每个 Release 的 `lunafox-<version>.zip` 包含等价的部署快照：

- `compose.yaml`、`.env` 和 `.env.example`
- 经过验证的最终 release manifest 和双 Registry 的 Engine inventory
- Loki 与 Alloy 配置
- 默认 fingerprint corpus 和所需 wordlists
- 七个 Bash 生命周期入口，以及共享的 `lunafox-lifecycle.sh` 辅助脚本

## 安装

### 从公共仓库安装

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
./install.sh --public-host luna.example.com
```

`install.sh` 恰好运行一次 `docker compose up -d`，并等待部署完全就绪。同样支持直接使用 Compose，这也是 Windows PowerShell 的原生入口：

```console
git clone https://github.com/yyhuni/lunafox.git
cd lunafox
docker compose up -d
```

### 从 Release 压缩包安装

从对应的 GitHub Release 下载 `lunafox-<version>.zip` 及其 SHA-256 元数据，按本地策略需要时完成校验，然后解压到永久目录：

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
./install.sh --public-host luna.example.com
```

解压后的 ZIP 也支持直接使用 Compose：

```console
unzip lunafox-<version>.zip -d lunafox-<version>
cd lunafox-<version>
docker compose up -d
```

### 设置公网地址

安装时用 `--public-host <host>` 和 `--public-port <port>` 设置 `PUBLIC_HOST` 与 `PUBLIC_PORT`，或沿用 `.env` 中已有的值。随包默认值是 `localhost` 和 `443`；内部 HTTPS `PUBLIC_URL` 由它们派生。

地址选项仅限安装动作：host 必须是 hostname 或 IP address，port 必须是 1 到 65535 的十进制值。未提供的值沿用 `.env` 或随包默认值，显式选项只会原子更新对应的 `.env` 键。所有选项在访问 Docker 之前完成校验。

### 选择镜像 Registry

两种安装来源在 .env 中都默认使用 Docker Hub：

```dotenv
RELEASE_REGISTRY=docker.io
```

如需使用 GHCR，请在首次启动前修改该值：

```dotenv
RELEASE_REGISTRY=ghcr.io
```

也支持自定义 HTTPS OCI Registry。产品镜像与 Engine Package / Runtime Image 可以使用不同来源：

```dotenv
RELEASE_REGISTRY=product.company.cn
ENGINE_INSTALL_REGISTRY=engine.company.cn:5443
```

`ENGINE_INSTALL_REGISTRY` 为空或未设置时默认使用 `RELEASE_REGISTRY`。填写 `host[:port]`，不带协议、路径、凭证或空白。镜像服务必须保留原 namespace 与 repository 路径以及不可变 digest；Engine 镜像同步还必须复制 OCI signature referrers 和 bundle。签名、包结构、平台、版本与兼容性校验继续生效。第一方内容只从配置的来源拉取，不会隐式回退到官方 Registry。预热、执行与升级使用相同设置，升级会保留独立的 Engine 设置。外部 Agent 也必须配置 `ENGINE_INSTALL_REGISTRY`。

该设置不会改写 PostgreSQL、Redis、Loki 或 Alloy 的地址；这些固定的第三方镜像仍使用宿主 Docker 配置。Registry 凭证与自定义 CA 是环境前置条件；该设置不会关闭 TLS，也不会替代 Sigstore trust-root 获取。原地升级成功后，产品镜像 override 仍绑定所选产品 Registry；更换该绑定需要保留应用数据与配置的显式迁移。

### 保护 .env 文件

一旦写入 `DB_PASSWORD` 或 `JWT_SECRET`，请用 `chmod 600 .env` 限制文件权限。生命周期脚本在进行任何变更前也会强制使用相同模式。配置输入只会作为 Compose secret files 传给 initializer，不得提交到仓库或打印到支持日志中。

## 首次启动与初始化

`docker compose up -d` 按以下顺序启动 one-shot services：

1. `config-init` 校验 `DATABASE_MODE`、派生的 `COMPOSE_PROFILES` 值和连接输入，生成或采用允许的 secrets，并将 database mode、database password 与 JWT secret 以 mode-0600 文件持久化到 `lunafox_config`。
2. `agent-preflight` 使用已发布的 Agent image，通过真实的 sibling-container round trip 校验 Docker API、精确的 named-volume identity、volume-subpath isolation 和只读 execution mounts。
3. `cert-init` 在 `lunafox_ssl` 中创建或校验证书和私钥。
4. 在 embedded 模式中，PostgreSQL 在 configuration initialization 之后启动，`migrate` 等待其 health。在 external 模式中，捆绑的 PostgreSQL 服务不存在，`migrate` 直接连接配置的 host。
5. `bootstrap` 等待 preflight 与 migration 完成，安装经过验证的 Engine inventory 和默认资源，然后创建或校验内部 Agent credential。
6. Server 只在 bootstrap 成功后启动。Agent 在 bootstrap 与 Server health 都成功后启动；Nginx 在 certificate、Server 和 Frontend readiness 之后启动。

报告就绪的启动即视为完成。可用 `docker compose ps` 查看进度。`docker compose config --quiet` 是可选的配置诊断；`docker compose pull` 是可选的预拉取步骤。

如果初始化失败，请检查 one-shot 与 Agent 日志：

```console
docker compose logs config-init agent-preflight migrate bootstrap cert-init agent upgrader
```

修正配置或明确修复命名 volume 状态，然后再次执行 `docker compose up -d`。

重复启动会复用 `lunafox_config` 中的文件。首次初始化成功后，改变 `DATABASE_MODE` 会在任何数据库 consumer 启动前失败。之后提供的非空 `DB_PASSWORD` 或 `JWT_SECRET` 如果与持久化值不同也会失败；Compose 不会轮换在线 credential。没有 mode record 的旧 configuration volume 默认采用 embedded mode；从此类 volume 采用 external mode 时，需要提供与持久化值匹配的明确 `DB_PASSWORD`。

默认资源遵循 fail-closed 规则。全新状态会一次性导入打包的 fingerprint corpus 和 wordlists。之后的 bootstrap 运行会保留已初始化的 wordlist 记录及其持久化内容（包括用户编辑），只校验 Catalog 身份和磁盘上的常规文件边界。默认 wordlist 缺失或被部分删除时会返回明确的修复错误；bootstrap 绝不会填补或覆盖这些状态。已经初始化但被完全清空的 fingerprint corpus 不会被静默重新 seed。

内部 Agent credential 是 `lunafox_agent_state` 中 mode-0600 的普通 JSON 文件；它绝不会通过主机环境或 Docker container metadata 传递。Registration、database binding 和 credential publication 是串行的。重复成功执行 `docker compose up -d` 会复用同一个已注册 Agent。缺失、格式错误、权限过宽、发布不完整、被删除或与数据库不一致的 identity state 都会返回明确的修复错误，绝不会静默注册另一个 Agent。

preflight 会在业务 bootstrap 之前检查实际 Docker socket、Linux execution node、named-volume subpaths 和只读 execution mounts。常驻 Agent 启动时会再次检查 socket、平台、API version 和精确 volume identities。不支持的能力会明确失败；不存在更弱的 whole-volume fallback。Server 永远不会收到 Docker socket。

Compose 管理的 Agent 禁用了容器内 self-update。受限 upgrader 会根据 release Manifest 更新固定的 `agent` service。现有远程 Agent 安装和 `update_required` self-update 行为不变。

## 登录与管理员密码找回

全新部署报告就绪后，打开 `PUBLIC_URL`，使用默认管理员账号登录：

| 用户名 | 密码 |
| --- | --- |
| `admin` | `admin` |

首次登录后，请前往账号设置及时修改密码。

如果忘记管理员密码，请在部署目录中运行：

```console
docker compose exec server resetadmin
```

该命令只重置现有的 `admin` 账号，只显示一次新密码，并使管理员现有会话失效。账号不存在时不会创建账号。

## 配置

### 数据库模式

`DATABASE_MODE=embedded` 是默认值，并启用捆绑的 PostgreSQL 服务。`DB_PORT=5432`、`DB_USER=postgres` 和 `DB_NAME=lunafox` 是可编辑的默认值。此模式下保持 `DB_HOST` 和 `DB_SSLMODE` 为空。`DB_PASSWORD` 为空时，会在首次初始化时生成。

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

`JWT_SECRET` 是可选项，为空时会生成。

<a id="database-environment-variables"></a>

## 数据库环境变量

`.env.example` 中的注释行是可选覆盖项。只有在需要替换默认值时才取消注释。下次启动会从 `.env` 读取修改后的值。

| 变量 | 未设置时的默认值 | 首次成功启动之后 |
| --- | --- | --- |
| `DB_HOST` | `postgres` | 可在之后的启动中修改。 |
| `DB_PORT` | `5432` | 可在之后的启动中修改。 |
| `DB_USER` | `postgres` | 可在之后的启动中修改。 |
| `DB_NAME` | `lunafox` | 可在之后的启动中修改。 |
| `DB_SSLMODE` | `disable` | 可在之后的启动中修改。External 模式在首次启动前仍必须显式设置。 |
| `DB_PASSWORD` | 在 embedded 模式中为空时生成。 | 固定。不同的非空值会失败，空值会复用已持久化的密码。 |
| `DB_TIMEZONE` | `UTC` | 可在之后的启动中修改。它设置应用连接使用的 PostgreSQL session time zone。 |
| `DB_MAX_OPEN_CONNS` | `25` | 可在之后的启动中修改。应用连接池的最大打开连接数。 |
| `DB_MAX_IDLE_CONNS` | `5` | 可在之后的启动中修改。应用连接池的最大空闲连接数。 |
| `DB_CONN_MAX_LIFETIME` | `300` | 可在之后的启动中修改。池化连接的最长生存时间，单位是秒。 |

首次成功启动后 `DATABASE_MODE` 即固定，`COMPOSE_PROFILES` 必须保持为 `${DATABASE_MODE:-embedded}`。`JWT_SECRET` 与 `DB_PASSWORD` 遵循相同的非空冲突规则。详见下文“首次启动后的配置变更”。

## 首次启动后的配置变更

首次成功运行 `config-init` 后，配置边界就会建立。它会将 `DATABASE_MODE`、`DB_PASSWORD` 和 `JWT_SECRET` 持久化到 `lunafox_config`；`COMPOSE_PROFILES` 必须继续由 `DATABASE_MODE` 派生。之后提供的非空输入都会与持久化状态进行校验。

本版本不支持以下任何一种原地操作：

- 在 `embedded` 与 `external` 之间在线迁移数据库模式。
- 对在线数据库密码或 `JWT_SECRET` 进行在线凭据轮换。
- 在 `lunafox_postgres` 与外部 PostgreSQL 服务器之间自动复制数据。
- 删除或替换 `lunafox_config`，以强制选择新配置。

简而言之，本版本不支持在线数据库模式迁移，也不支持在线凭据轮换。

如果已有部署因误改配置而被阻塞：

1. 在 `.env` 中恢复原来的 `DATABASE_MODE` 和 `COMPOSE_PROFILES`。
2. 恢复原来的非空 `DB_PASSWORD` 和 `JWT_SECRET`，或者删除新填入的值，使空输入复用持久化文件。
3. 保留 `lunafox_config`；embedded mode 还要保留 `lunafox_postgres`。external mode 则要保留外部数据库及其由运维管理的备份。不要执行 `docker compose down --volumes`。
4. 运行 `docker compose up -d`；如果初始化仍然失败，在修改任何持久化状态前先检查 `docker compose logs config-init`。

如需采用其他模式或其他凭据，请使用独立的新部署：

1. 保留旧部署，并对 embedded mode 的 `lunafox_config` 和 `lunafox_postgres`，或 external mode 的外部数据库，分别执行已验证的备份。
2. 创建新的部署目录，在第一次 `docker compose up -d` 前写入最终的 `.env`、`DATABASE_MODE`、`COMPOSE_PROFILES`、连接参数和 secrets。
3. 如果必须保留数据，请使用运维方已验证的 PostgreSQL backup/restore 或 export/import 流程。在切换流量前验证 schema、应用数据、连接访问和 health。LunaFox 不会复制数据、修改远程 role，也不提供自动 rollback。
4. 新部署验收完成前，保留旧部署作为 rollback 目标。

凭据轮换仍需在此 Compose workflow 之外协调完成。PostgreSQL 密码变更必须与数据库 role 和新部署的 `DB_PASSWORD` 同步；变更 `JWT_SECRET` 可能使现有 session 失效。不要直接编辑 `lunafox_config` 中的文件。

## 日常运维

### 生命周期脚本

每个快照都会在部署根目录提供七个 Bash 入口，以及共享的 `lunafox-lifecycle.sh` 辅助脚本。它们封装 Compose，补充 Docker 无法表达的检查，并且不会维护第二套部署状态：

| 脚本 | 等价 Compose 命令 | 行为 |
| --- | --- | --- |
| `./install.sh` | `docker compose up -d` | 首次启动和安全重跑。`--public-host` 和 `--public-port` 只更新对应的 `.env` 键；命名 volume 与其他设置保持不变。 |
| `./start.sh` | `docker compose up -d` | 启动已有部署。 |
| `./restart.sh` | `docker compose up -d --force-recreate` | 将当前 `.env` 和 `compose.override.yaml` 应用到每个容器。 |
| `./stop.sh` | `docker compose stop` | 停止所有常驻服务并保留全部数据。 |
| `./status.sh` | `docker compose ps` plus the readiness checks | 只读摘要；只有完全就绪时才返回 0。 |
| `./logs.sh` | `docker compose logs --tail 200 --follow` | 只读日志访问，并转发服务名称和 Compose 选项。 |
| `./uninstall.sh` | `docker compose down --remove-orphans` | 移除容器和项目网络；默认保留 volumes、`.env` 和 `compose.override.yaml`。 |

只有在 one-shot tasks、核心服务健康状态、辅助服务稳定性、常驻 Agent 的 claim-ready state 和公共 HTTPS endpoint 全部通过后，`install.sh`、`start.sh` 和 `restart.sh` 才报告成功。`install.sh` 允许十五分钟，另外两个允许五分钟；`LUNAFOX_READY_TIMEOUT_SECONDS` 可以为单次调用覆盖此窗口。

失败时会打印 `FAILED`，指出失败的服务或检查，并保留部署现场：容器、volumes 和配置保持 Compose 创建后的状态，不会回滚。诊断输出顺序固定为结论、阶段、保留现场、后续命令；如果已经识别服务，会指向 `./logs.sh <service>`，否则指向 `./logs.sh`。

`install.sh` 会描述它识别到的运行类型，而不是始终叙述为首次启动：没有任何 LunaFox 数据的目录属于首次启动；只有 volumes 而没有容器的目录会在保留命名 volumes 和已安装 Engine 的情况下重建容器，不属于升级；运行中的部署会在保留配置和 volumes 的情况下启动。当这种重建部署在 `bootstrap` 等首次启动任务上失败时，footer 会在 `logs.sh` 和 `status.sh` 提示之后增加一行可选信息：如果可以丢弃保留数据，只有 `./uninstall.sh --purge --confirm` 能删除命名 volumes 并重新开始。

`install.sh` 接受 `--public-host <host>`、`--public-port <port>` 和 `--help`。公网地址参数仅限安装动作：host 必须是 hostname 或 IP address，port 必须是 1 到 65535 的十进制值。未提供的地址值沿用 `.env` 或随包提供的 `localhost` 和 `443` 默认值；显式参数只会原子更新对应的 `.env` 键。命令会在访问 Docker 前校验参数，并且恰好运行一次 `docker compose up -d`。

默认的 `uninstall.sh` 是可恢复的：它保留命名 volumes、`.env`、发布目录，以及已完成 upgrade 写入的常规 `compose.override.yaml`。`./uninstall.sh --purge --confirm` 会按当前 Compose 文件栈（`compose.yaml` 和存在的常规 `compose.override.yaml`）声明的精确名称永久删除已存在的 volumes，忽略缺失或不同的归属标签，包括中断安装留下的无标签残留卷。随后移除该版本覆盖层，以便在同一目录中进行干净重装；未声明卷、`.env` 和发布目录仍会保留。

确认 purge 后会跳过安装用的归属与数据库模式分类。若 `.env` 缺失，只读使用 `.env.example` 解析 Compose，不会创建 `.env`；无效的 Compose 输入仍会阻止清理。移除本项目容器后，只要任一目标卷仍被容器引用，就不会删除任何卷。Docker 查询或删除失败会报错并保留版本覆盖层；删除过程中失败时，较早处理的卷可能已经被删除。已无目标卷时允许重复 purge。确认 purge 即授权按声明的精确名称删除，即使卷属于其他项目，其数据也无法通过此命令恢复。

### 日志

所有常驻服务都使用 Docker 的 `json-file` driver，设置 `max-size=10m` 和 `max-file=3`。`./logs.sh` 会以相同的有界输出跟随所有服务；传入服务名称或 Compose logs 选项可以缩小范围，例如 `./logs.sh server`。普通 Alloy 容器只通过 Docker socket 读取带有 LunaFox Server 和内部 Agent 标签的输出，并将其发送到本地 Loki 服务。Alloy 将读取位置保存到 `lunafox_alloy`，因此 collector 重启不会重新播放全部保留日志。

collector 保留现有 selectors `{component="server",container_name="lunafox-server"}` 和 `{agent_id="<id>",container_name="lunafox-agent"}`。它不会采集自己的输出。不安装也不需要 Loki Docker plugin。

### 镜像预热与恢复

所有现代安装入口都使用同一个由 Compose 管理的 `engine-preheater` one-shot gate：公有仓库、Release ZIP、直接 Compose，以及仍支持的开发与私有入口，都会消费同一份 release-bound `preheat-manifest.json`。应用服务启动前，它会预热当前宿主平台发布的全部 Engine Runtime，以及选定 Compose profile 的完整镜像闭包。即使某个 Workflow 被关闭，已发布的 Engine Runtime 也会保留在预热范围内；`embedded` 包含 `postgres`，`external` 排除它。

预热 deadline 与 readiness 相互独立。`LUNAFOX_PREHEAT_TIMEOUT_SECONDS` 默认是 `900` 秒，允许 `300` 到 `3600`；`LUNAFOX_READY_TIMEOUT_SECONDS` 只控制外层 readiness 等待。如果 readiness 先超时，预热器会继续运行。预热失败会保留容器、日志、已验证 Docker cache、命名 volumes、数据库和应用数据。修复传输或配置问题后，可先查看只读 status 和服务日志，再只重跑这个 gate：

```console
docker compose up -d --force-recreate engine-preheater
```

执行升级时，目标 Compose 和 `third-party-image-policy.json` 从 immutable `manifests/<release-tag>/` channel 目录获取，并在私有 candidate snapshot 预热前与该 release 的 preheat manifest 校验绑定；只有 gate 成功后才会提升活动部署。

### 部署锁与恢复

生命周期变更和升级绝不会重叠。两者都会获取部署级锁 `.lunafox-lifecycle.lock`，而 `./status.sh` 会不等待地报告该锁：

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

`down` 会移除容器和项目网络，同时保留命名 volumes，包括 `lunafox_upgrade_state`。除非明确要删除并已备份永久状态，否则不要添加 `--volumes`。本部署中的任何命令都不会自动迁移、重置或删除旧部署。

## 系统更新

发布包会在 `compose.yaml` 中固定其 `stable` 或 `canary` channel 和 public metadata source。没有主机 update script：版本变化通过 Upgrade Operation 和受限 Compose upgrader 完成，生命周期脚本既不复制也不绕过该状态机。所选 Registry 来自首次启动前的 `.env`，之后由升级保留。

Server 通过 HTTPS 获取当前 schema-v3 channel record，校验其受限的 Manifest path 和原始 SHA-256，然后按 digest 将不可变 Manifest 缓存到 `lunafox_upgrade_state`。同时，它会从按版本隔离的 `manifests/<release-tag>/runtime-composition.json` 路径下载 manifest 绑定的 composition，并将已验证字节保存到 `.lunafox/upgrade/compositions/<composition-core-digest>.json`。运行中的 Server binary 仍是 `upgrade.compatibilityRange` 的输入；host 的 confirmed deployment inventory 才是 candidate 可用性和组件比较的 baseline，因此已确认的 frontend-only release 不会再次被提供。

管理员可以通过现有 frontend update control 检查、确认并启动符合条件的更新。在 Server 暂停调度或取消 work 之前，支持 v2 的 host 会返回一个只读 scope plan，将 candidate manifest、composition、confirmed baseline 和 live container digest 绑定在一起。只有确认过 dynamic frontend upstream 能力、且差异严格为 `{frontend}` 时，才会产生 `frontend_only`；它只在无 dependencies、无 build 的情况下 pull 和 recreate `frontend`。deployment lock 获取后会重新验证该 plan；stale plan 会在副作用前失败，绝不会自行扩大为 `full`。Server 只通过共享 Unix Socket 发送 Operation ID、固定 action 和 Manifest digest。即使浏览器关闭或 Server 被重建，Compose 管理的 upgrader 仍会继续。它没有 network port，并使用固定的 project、file set、command argv、service allowlist 和 `--no-deps` recreation。Server 永远不会收到 Docker Socket。

upgrader 的可写 Docker Socket 赋予它控制本地 Docker daemon 的能力。该服务还挂载解压后的 package directory，以便读取 `compose.yaml` 和 `.env`。full operation 会在完成 service、migration、Agent、health 和 digest 校验后原子安装 `compose.override.yaml`；`frontend_only` operation 只暂存修改 frontend image 的 patch，只有 Server 发送 `confirm` 后才晋级持久化 override。请将该 override 与解压目录一起保留：普通的 `docker compose up -d`、`start` 和 `restart` 会自动保留已确认的 image 和 version target，不会重写 database、public address 或 secret 设置。

host 只有在完整验证后才会替换 confirmed deployment state。对于 `frontend_only`，只有 frontend digest、health 和 public Nginx response 收敛后才会记录新的 component inventory；失败或部分执行绝不会被猜测为新的 baseline。

完成 receipt 只证明受限的 Compose 部署工作结束。Server 会将它与 database Operation、journal、migration、service、API 和 Agent evidence 对账后才报告成功。migration 失败或结果不确定时需要恢复；Agent 超时需要人工处理。当前升级策略使用冻结的 `000001` migration baseline，仍不提供保留数据的 rollback 或自动备份。生产恢复只能使用已验证的备份恢复或批准的前向修复；migration `down` 文件仅用于测试 teardown。

明确只支持 `schema-v1` 的 host 沿用既有 `full` 路径；缺少 composition cache 也只允许 `full` planning。损坏或篡改的 composition、无效的 v2 capability/plan 数据以及 stale scoped plan 会在 scheduler、cancellation 或 Compose 副作用前 fail closed，不会静默降级。声明 migration 或修改任何非 frontend 组件的 candidate 同样保持 `full` 路径。

自动更新支持兼容的 image-only releases。改变 Compose 结构、named volumes、initialization resources 或 upgrader protocol 的 release 必须声明不兼容；请下载新的 deployment ZIP，保留文档说明的状态，检查其 README，并从新的解压目录运行 Compose。运行中的 release 必须满足 Manifest 的 `upgrade.compatibilityRange`；否则更新仍可见，但不能创建自动 Upgrade Operation。如果主机无法向 upgrader container 暴露本地 Docker Socket，这也是手工 fallback。

当前迁移策略仍处于 `release-candidate` 阶段：`000001` 基线已冻结，保留数据升级尚未开放。因此 candidate 只要声明 `hasDatabaseMigration: true`，更新检查仍会显示目标版本，但 `eligible` 为 false，应用内无法创建 Upgrade Operation，界面会提示按部署指南手动升级。此时请按下面的手动路径处理：先按“备份与恢复”备份 `lunafox_config` 与 `lunafox_postgres`（external mode 按其已批准流程备份），保留 `.env` 中的非空凭据和命名 volumes，下载目标版本的 deployment ZIP，阅读其中的 README，从新的解压目录运行 `./install.sh` 或 `docker compose up -d`。新版本 Server 启动时会应用编号 forward migration；完成后用 `./status.sh` 核对 readiness，并确认登录与数据仍然可用。建议把本部署指南交给 AI 助手（如 Codex）逐条执行并核对每一步输出；不要跳过备份，也不要手工改写 schema 或 migration 表。

一个经过显式登记的兼容 bridge 可以省略两个现代 Manifest metadata block，使旧版严格客户端仍能读取目标。这个例外只对精确版本生效，仍保留同一套不可变 image 和 upgrade inventory，并且只能走完整的 Compose 更新路径，绝不会产生 `frontend_only` plan。其 Release 仍会携带独立的 release notes 和 composition evidence asset，但旧 host 在升级时不会下载没有 Manifest binding 的 composition。这是直接升级到当前发布版本，不要求用户先安装一个中间版本。

## 证书

`cert-init` 会创建带有 `PUBLIC_HOST` DNS 或 IP SAN 的自签名 RSA 证书。已有证书文件必须是权限受限的普通文件、未过期、与配置的 host 匹配，并且与私钥成对。部分或无效状态会失败，而不是被覆盖。本版本不提供 ACME、用户证书导入或自动续期。

Nginx readiness probe 使用 `127.0.0.1` 以匹配其 IPv4 listener，即使在 `localhost` 优先解析为 IPv6 的主机上也是如此。Upgrade journal 和完成 receipt 文件只存在于 `lunafox_upgrade_state`，并由 Upgrade Operation API 对账。

## 备份与恢复

对于 embedded mode，请将 `lunafox_config` 与 `lunafox_postgres` 一起备份。对于 external mode，请按照 operator 批准的流程，将 `lunafox_config` 与外部数据库一起备份。只删除 configuration volume 会丢失连接所选数据库所需的 password。

从只把 credential 存在 `.env` 的包升级时，新包首次启动仍需保留已有的非空 `DB_PASSWORD` 和 `JWT_SECRET`。`config-init` 会将它们复制到 `lunafox_config`。完成迁移后，空输入会复用持久化文件。冲突值会明确失败；本版本不支持在线凭据轮换。

在 embedded 与 external PostgreSQL 之间迁移现有安装不属于原地操作。独立部署与运维数据迁移路径见“首次启动后的配置变更”。

## 故障排查

- readiness 超时：运行 `./status.sh` 和 `./logs.sh <service>`；失败 footer 会指出失败阶段，且不会回滚任何内容。修复原因后重复同一命令。
- 预热失败：检查 `engine-preheater` 的 status 和日志，修复传输或配置问题，然后用 `docker compose up -d --force-recreate engine-preheater` 只重跑该 gate。
- 浏览器提示证书警告：首次启动会为 `PUBLIC_HOST` 使用自签名证书；本版本不提供自动证书管理。
- 主机端口被占用：在首次启动前设置其他 `PUBLIC_PORT`，或释放该端口。
- 修改后数据库凭据冲突：恢复原来的非空 `DB_PASSWORD` 和 `JWT_SECRET`，或清空它们，让空输入复用持久化文件。
- 部署锁存在：确认没有生命周期命令或升级正在运行，然后手工删除 `.lunafox-lifecycle.lock`。
- 升级进入 recovery 或 attention：以 `lunafox_upgrade_state` 中的 upgrade journal 为事实来源并走恢复路径；不存在自动 rollback。

## 发布与安全边界

这是单节点 Compose 部署，不提供滚动升级、自动备份、保留数据的 rollback 或跨节点恢复。已发布的 `000001` migration baseline 不可变，也不承诺与已有持久化部署兼容。Release 生成和契约检查使用隔离 fixture，不会触碰操作者的 volumes。

当前 Agent authentication credential 仍是长期有效的八字符十六进制 bearer。本版本不提供 Agent TLS certificate-chain identity、replay protection、rotation 或 revocation。TLS 和 credential hardening 需要单独的安全变更。本 package 用于由操作者拥有或控制的基础设施上的 self-hosted 场景；封闭 Agent artifact 仍受 `NOTICE-CLOSED-ARTIFACTS.md` 约束。闭源 Agent artifact 的再分发或商业使用需要事先取得书面授权或商业协议。

导出的第一方 Runtime 源码和部署文件采用 SPDX `GPL-3.0-only` 许可。

Engine Runtime Images 和 Engine Packages 是公共 artifact。第一方 Engine source 采用 GPL-3.0-only。随附的第三方组件继续受其适用的许可证和归属条款约束。
