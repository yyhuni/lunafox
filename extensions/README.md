# Extensions

`extensions/` 是 LunaFox 的构建期扩展定义源码边界，不是 Server、Agent 或 Engine Container 的运行时配置/加载目录，也不用于存放普通第三方依赖、vendor 代码或临时文件。

## 当前目录

- `workflows/`：随发行版提供的 scan workflow 定义源码。部署时复制或只读挂载到 `WORKFLOW_DEFINITIONS_ROOT`，Server 只读取部署后的目录。
- `engines/`：第一方 Engine authoring 源，包括 strict `engine.v5`、locales、唯一生成 Facade、Engine-owned Go module、Runtime Image Dockerfile、扫描业务代码和测试。发布链先产出已验证 Runtime Image，再生成 exact Engine Package v2；运行时不得直接读取该目录。
- `engines/fingerprint_detection/`：独立的 Fingerprint Detection Engine。它只消费平台交付的 FingerprintHub Web resource，并在 pinned Observer Ward Runtime Image 中生成 current-only Website technology observations；其规则格式、空库 fail-closed 行为和升级重验要求由该 Engine 的 README 约束。

## 边界规则

- Server、Agent 和已构建 Engine Container 的生产路径不得通过仓库路径直接读取 `extensions/`；Engine business source 只能通过自身 module 的编译期依赖引用近目录代码。
- 只有构建、发布和部署工具可以读取这里的源码，并将其转换或复制为明确的运行时制品。
- workflow 运行时只认 `WORKFLOW_DEFINITIONS_ROOT`；目录缺失、定义无效或引用的引擎未安装时必须失败，不得回退到 `extensions/workflows/`。
- Server 只从已安装 exact package 编译并持久化执行计划；Agent 只消费 saved plan 和授权 artifact streams；Engine 只消费 canonical Context、挂载文件和 workspace。三者都不得以 `extensions/engines/`、runtime manifest、config-schema artifact 或 source-tree `package.json` 作为 fallback。
- 第一阶段扩展集合只包含稳定 `engine.lunafox.*` 身份的 LunaFox 第一方 Engine。第三方 publisher/identity/trust onboarding 和非同等信任 Engine 的 Server-enforced result capability policy 必须通过独立 OpenSpec security change 定义，不能由 typed Results 或 manifest allow-set 隐式授权。
- 新增扩展类型时，必须在最近的 README、schema 或契约中定义源码格式、产物格式、部署位置、加载方和验证入口。
- 生成物和本地构建输出不得写回 `extensions/`；应进入既有的 `dist/`、安装目录或发布制品存储。

## 验证

- 工作流定义或部署边界：`make verify-boundary-contracts`
- 工作流加载行为：`cd server && go test ./internal/scanworkflow/...`
- 涉及发行镜像或制品装配：`make verify-release-contract`
- 引擎定义与 engine-author 边界：`cd extensions/engines && go test ./...`

工作流 loader 和部署路径的机器门禁位于 `scripts/ci/boundary-contracts/workflow-definition-boundaries.mjs`；引擎定义与 engine-author source lane 门禁位于 `scripts/ci/boundary-contracts/engine-boundaries.mjs`。
