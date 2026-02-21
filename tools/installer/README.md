# LunaFox Installer

LunaFox 安装器以 Go 实现，默认使用终端交互向导完成安装，不再提供 Web 安装页面。

## 命令矩阵

- `./install.sh`
  - 用户安装入口（远程发布产物）
  - 支持：`--version`、`--channel`、`--source`、`--public-url`、`--public-host`、`--public-port`、`--non-interactive`
- `./scripts/dev/install.sh`
  - 开发者调试入口（本地源码）
  - 支持：`--public-url`、`--public-host`、`--public-port`、`--non-interactive`
- `./start.sh` / `./restart.sh`
  - 仅生产模式
- `./scripts/dev/start.sh` / `./scripts/dev/restart.sh`
  - 仅开发模式
- `./stop.sh`
  - 同时停止 prod/dev
- `./uninstall.sh`
  - 默认完全删除
  - `--keep-data` 保留数据

## 标准安装入口（发布产物）

```bash
./install.sh
```

`install.sh` 流程：
1. 读取 `stable` 通道（或你传入的 `--version`/`--channel`）。
2. 从远程 release 拉取对应平台安装器二进制。
3. 校验安装器 SHA256，并读取 `AGENT_IMAGE_REFS`/`WORKER_IMAGE_REFS`（digest 候选列表）。
4. 启动终端向导（或非交互模式），执行安装步骤。

常用示例：

```bash
./install.sh --help
./install.sh --version v1.5.13
./install.sh --channel stable --source github
./install.sh --channel canary --source github
./install.sh --public-url https://10.0.0.8:8083 --non-interactive
./install.sh --public-host 10.0.0.8 --public-port 8083 --non-interactive
```

说明：
- `install.sh` 不支持 `--dev`，开发调试请使用 `./scripts/dev/install.sh`。
- `install.sh` 不支持 `--` 参数透传。
- `install.sh` 不再支持 `--web`、`--listen`。
- 下载源策略：`--source auto` 时按 `GitHub -> Gitee` 顺序重试。
- 发布通道清单当前 schema 为 `SCHEMA_VERSION=2`，并强制 `AGENT_IMAGE_REFS`/`WORKER_IMAGE_REFS` 使用 digest 列表。
- `SCHEMA_VERSION=2` 下不再接受旧单值键 `AGENT_IMAGE_REF`/`WORKER_IMAGE_REF`。
- 安装器路径锚点采用单一真相：`--root-dir` 必填，不再读取 `LUNAFOX_INSTALLER_ROOT_DIR`，也不再自动按 `cwd` 探测。
- `--root-dir` 必须是有效项目根目录（需包含 `docker/docker-compose.yml`、`docker/docker-compose.dev.yml`、`tools/installer/cmd/lunafox-installer/main.go`）。
- 安装开始时会输出 `ROOT_DIR`、`DOCKER_DIR`、`ENV_FILE`，用于快速确认路径正确性。

## 本地调试入口（源码运行）

```bash
./scripts/dev/install.sh
./scripts/dev/install.sh --public-url https://10.0.0.8:8083 --non-interactive
```

等价底层命令：

```bash
cd tools/installer
go run ./cmd/lunafox-installer --root-dir /path/to/lunafox --dev
```

## 交互与非交互规则

- 默认行为：
  - 在交互终端（TTY）下，如果未显式传入公网地址，会进入终端向导。
- 非交互行为：
  - `--non-interactive` 模式下必须显式传入地址。
  - 在非 TTY 环境（CI/管道）下，若未传地址会直接失败。
- 地址参数：
  - 主机仅支持 `localhost` 或 IPv4，暂不支持域名和 IPv6。
  - 二选一：
    - `--public-url https://10.0.0.8:8083`
    - `--public-host 10.0.0.8 --public-port 8083`

## 发布通道约定

- `stable`: 正式版 tag（`vX.Y.Z`）自动更新。
- `canary`: 预发布 tag（`vX.Y.Z-rc.N`、`-beta.N`、`-alpha.N`）自动更新。

## TLS 行为（严格模式）

- 安装器默认启用严格 TLS 校验，不再使用不安全跳过校验。
- 首次安装会按公网主机生成 SAN 证书（包含 `publicHost`、`localhost`、`127.0.0.1`）。
- 健康检查与 Agent 注册 API 共用 `docker/nginx/ssl/fullchain.pem` 作为信任链。
- 若证书不匹配公网主机或证书链异常，安装会直接失败并提示修复。

## 标准发布流程（推荐）

1. 本地先验证：
   - `./scripts/dev/install.sh`
2. 发布测试版（canary）：
   - `git tag v1.6.0-alpha.1`
   - `git push origin v1.6.0-alpha.1`
3. 在测试机器验证分发：
   - `./install.sh --channel canary`
   - 或 `./install.sh --version v1.6.0-alpha.1`
4. 发布正式版（stable）：
   - `git tag v1.6.0`
   - `git push origin v1.6.0`
5. 用户默认安装正式版：
   - `./install.sh`

这套流程的关键点是：测试版和正式版通道隔离，`stable` 用户不会自动拿到 `canary` 版本。
