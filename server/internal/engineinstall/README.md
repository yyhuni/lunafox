# engineinstall

安装边界只负责匿名 HTTPS OCI Distribution 传输、descriptor/digest/size 与运行兼容性；包内
`engine.json`、`package.json`、locale 和版本校验必须委托给
`contracts/enginemanifest`，不得在此复制；legacy runtime manifest/bundle 字段和布局
由 contracts 统一拒绝。Package v2 只接受 contracts 定义的封闭四文件 layout，
不在本模块维护第二套包内 schema。

- OCI client: `oras.land/oras-go/v2`。生产代码不得调用 `oras` CLI。
- 发布者签名可作为第一方发行流水线的证据保留，但通用安装器不解释它，也不以它、
  GitHub OIDC、Registry allowlist、Engine namespace 或 repository 命名决定准入。
- 普通 Server 的运维安装与 production bootstrap 必须保持匿名 HTTPS；它们不接受
  plain HTTP、任意 Registry transport 重写或单平台 Runtime Image。唯一例外是显式
  `ENGINE_INSTALL_CF_ACCELERATION=true`：它只能将已通过 GHCR Cosign 验签的第一方
  `yyhuni/lunafox-*` 同一 digest 下载依次指向 CF、Docker Hub、GHCR，不能作为新的
  发布身份或通用 Registry 重写。CF 的 DNS/TCP/TLS/超时/429/5xx 故障可继续尝试后续
  候选；CF 401/403/404、OCI 完整性、协议错误和 GHCR 验签失败必须终止。仅显式
  `ENGINE_INSTALL_DEVELOPMENT_MODE=true` 的 disposable local Compose bootstrap 可以同时
  开启 plain HTTP、单 candidate inventory、当前宿主单平台 index，以及一组精确的 Runtime
  Image identity-to-transport Registry authority。四项开发配置必须完整，否则在 inventory
  安装前 fast-fail。该明确的开发 bootstrap 也是其已验证 inventory 的替换调用方，因此重复
  本地发布可将同一 Engine ID 的 current package 更新为新 digest；production bootstrap 和
  interactive 安装仍必须遵守显式 replacement 规则。
- development Registry 映射只改变 `RuntimeImageIndexVerifier` 的远端连接 authority；package
  中的 canonical Runtime Image ref、digest、verified identity 和持久化 registration facts
  必须保持原值。映射不匹配时不得回退到原 Registry 或从 ref 推断 transport。
- 依赖升级由 Go module 的常规安全更新流程管理；升级 ORAS 或 Cosign 时必须重新运行
  Registry pull、signature failure 与 integrity fallback 测试。

## Package v2 installation lane

该实现是 Server-only 的唯一 Engine Package 安装 lane：

- `LoadInventory` 只读取 bootstrap 所需的有序 immutable artifact refs。默认 production
  每个条目恰好两个同 digest 候选；启用 CF 加速的 production 条目必须恰好为 CF、Docker
  Hub、GHCR 三个同 repository/digest 候选；显式 local development 每个条目恰好一个候选。
  顺序只表示传输 fallback，Engine identity 必须从已验证的 package 内发现。
- `ORASPackagePuller` 从 canonical `artifactRef` 校验 OCI manifest descriptor、完整
  manifest bytes、显式 OCI image-manifest root media type、精确 OCI empty config
  descriptor、v2 artifact/layer media type 和唯一 package layer descriptor，并在
  consumer 回调内完成 layer 流读取。缺少 root media type 或使用 OCI image config 的旧
  envelope 必须重新发布并更新 artifact reference；它们在 package layer 拉取前 fail
  closed，且绝不前进到下一 Registry candidate。
  只有明确的候选可用性或远端流错误可以前进到下一候选；签名、完整性、调用方预算、本地
  处理和未知错误立即失败。
- `CacheInstaller.StageAndPromotePackage` 流式校验 descriptor size 和完整 archive
  bytes 的 `packageDigest`，安全解包并在 package 内 identity 与 package-bound Runtime
  Image 验证成功后，才原子提升 archive 和只读 expanded derivative。
- `RuntimeImageIndexVerifier` 按 package 声明顺序验证真实 digest-qualified OCI image
  index；CF 加速模式先验签原始 GHCR digest，再从 CF 开始传输候选；production 每个候选必须
  包含唯一 `linux/amd64`、`linux/arm64` platform
  descriptors，显式 local development 必须只包含当前受支持宿主平台中的一个。两种模式
  都不得把所选 ref 或 `runtimeImageDigest` 建成独立持久身份。
- `EnginePackageInstaller` 串联上述边界，只返回后续 repository adapter 所需的
  Engine ID、publisher、package version、canonical artifact ref、`packageDigest` 和
  verified manifest projection。
- `EngineRegistrationService` 只在完整验证和 cache 提升成功后，把这六项事实及
  当前 Engine 记录交给 context-aware repository 事务；上游失败、取消或数据库
  失败不会创建 pending/failed 伪注册。

`CacheInstaller.LoadOrRebuildPackage` 仍是只处理既有 digest-keyed archive 的加载/
重建 seam。每次使用前先校验完整 archive bytes；expanded entry 的结构、权限或语义
任一失效或四文件内容不再与 archive 精确一致时，必须把整个 entry 移出 canonical
path，再从已验证 archive 安全重建、校验并原子提升为只读目录。

`installedengines.RepositoryBackedQuery` 读取一次数据库 current record，并仅用该
record 的 canonical `packageDigest` 通过 `CacheInstallerExactPackageLoader` 加载/
重建精确 cache entry；返回值把该 record snapshot 与 path-free v2 layout 绑定。任何
archive、source、artifact ref、Engine ID、publisher、package version 或 manifest
projection 漂移都 fail closed，不扫描 sibling、不拉取 Registry，也不改写注册。

Package v2 已是 Server 安装、cache 和 catalog 的唯一生产契约；不向 Agent 分发
package。Agent 和 Engine runtime 不得调用 package decoder、cache 或 materializer；
已删除的 Worker、v1 package/runtime manifest、runtime bundle 和 HostCommand 路径不得恢复。

## 运维授权边界

当前每个已认证用户都是安装授权主体。用户提交一个 canonical `registry/repository@sha256:...`
后，Server 必须校验 OCI envelope、Package v2、package/manifest identity、Runtime Image
digest/platform 与 cache；这些完整性检查不能跳过。`publisher` 只是包内声明的展示元数据，
`engine.lunafox.*` 也不拥有特殊安装权限。tag、URL、上传、plain HTTP、私有 Registry
凭证和 workflow 导入不在当前范围。

同一 `engineId + packageDigest` 安装是幂等的。已有 Engine ID 但 package digest 不同
时，默认冲突；调用者必须明确提交 `allowReplacement=true` 才能原子替换 current record。
安装不会创建或改变 workflow。`WORKFLOW_DEFINITIONS_ROOT` 只在 Server 启动同步内置
release definition 时读取；只有持久化 workflow 明确引用该稳定 Engine ID，才会使它进入扫描编排。

已安装 Engine 可以通过现有 task/session scoped typed Results 边界提交平台支持的结果类型。
Agent session 认证、Server schema、内容和 target scope 校验仍然是授权边界；package
不声明 result allow-set。

## Migration and rollback boundary

当前安装链只面向可清空数据库、cache、volumes、证书和配置后重新安装的 disposable
development，不提供 preserve-data upgrade。开发期回退只能从 Git/分支历史恢复代码，
随后执行空状态 fresh install；不能为回退保留 v1、Worker、runtime bundle、兼容 decoder
或数据库 migrate-down 路径。

当前 `000001` fresh-install schema 仅可在首次公开稳定版本或首次必须保留数据的
non-disposable 部署出现前继续更新/squash；两个触发点以先到者为准冻结 baseline。
冻结后不得改写已发布 migration，数据库只能通过新的 numbered forward migrations
演进，升级流程不得执行任何 migrate down。package installer 不是数据库升级或回滚入口。
