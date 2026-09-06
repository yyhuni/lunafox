# Nuclei POC 模块

本模块负责 Nuclei POC 的公开源目录管理，并向 Server 的执行 Artifact resolver
提供一个窄的、只读的运行时投影。目录管理和扫描执行仍然是两个边界：本模块不
启动 Nuclei、不读取 Agent 会话，也不决定 Workflow 是否启用。

- 当前只有一个全局生效源；同步任务先写候选集合，完整校验成功后在一个事务内整体替换旧目录。
- `templateId` 是对外唯一身份，`sourceId` 只作为内部关联，为后续多源演化保留；任务、源和候选记录都必须保持一致关联。
- 原始 YAML、相对路径和可查询元数据由 Server 持久化；列表不返回 YAML，详情才返回内容。
- `GET /v1/nucleiPocs/filterOptions?field=tags` 是受保护的标签选项投影；它只读取完整已提交目录的 `tags` 列，按小写规范化、每个 POC 内去重并返回目录范围计数，结果按标签值稳定排序。缺少 `field`、重复参数或请求其他字段都会返回 `INVALID_ARGUMENT`，不会退化为分页列表。
- `isEnabled` 是独立的持久化运行时覆盖：匹配的 `templateId` 在连续同步时继承，上一份目录中不存在的模板（包括删除后重新出现的模板）默认关闭。当前开发基线通过重建 schema 切换默认值，不对既有数据库做回填。
- `POST /v1/nucleiPocs:setActivation` 在现有认证全局边界内对完整已提交目录设置显式 `enabled` 目标，并返回事务内实际变化的 `affectedCount`；分页、搜索和筛选不会缩小范围。空目录和重复目标都是成功的零变更，非终态同步会先返回 `SYNC_ALREADY_RUNNING` 且不写入任何 POC。
- 同步任务创建、最终 promotion、单条启停和全目录启停共享短暂的 catalog mutation 边界：进程 mutex 加 PostgreSQL transaction-scoped advisory lock；clone、遍历、解析和候选 staging 不在锁内。全目录启停的列表/详情缓存由前端刷新，source metadata 保持不变，网络结果未知时不会自动重试。
- 同步通过持久化异步任务运行，失败、过期、残留清理和幂等请求均由本模块处理；失败不得改变上一份已提交目录。
- 源校验只负责匿名 HTTPS URL 形状，不执行 DNS、公网 IP、直连或重定向预检；`custom` 源的实际出口由 Git 和部署代理决定，因此可使用 Fake-IP 或内网目标。
- Git 按域名连接并继承 `HTTP_PROXY`、`HTTPS_PROXY`、`ALL_PROXY`、`NO_PROXY` 及其大小写变体；仍不支持私有凭据、SSH、HTTP、`git://`、本地路径或自动同步。
- Docker Compose 的开发和生产 `server` 服务都会传入这些代理变量；代理地址必须从 Docker VM 可达（例如 `host.docker.internal`），无代理环境保持为空。
- 同步会遍历仓库中的 YAML/YML 文件；不能解析为有效 Nuclei 模板或出现重复 `templateId` 的文件会跳过，其他有效模板仍会继续导入。路径逃逸、符号链接和资源配额违规仍会终止任务；若没有任何有效模板，任务以空候选失败并保留有界诊断。
- YAML 解析后的元数据若包含真实 NUL 字节，会在持久化投影中编码为文字 `\u0000`；原始 YAML 内容和内容哈希保持不变，`templateId` 仍严格拒绝 NUL。

## 运行时模板选择

`ListEnabledForExecution` 只在 Nuclei Task 进入准备阶段时调用，读取当前事务可见
的 `isEnabled=true` 行并按 `templateId ASC` 排序。Scan 创建、保存的 execution
plan 和 Workflow 都不冻结模板集合；因此长期排队的 Task 可能在启动时使用后来启用
或更新的模板。Server 在一次 `ExchangeRuntimeArtifact` 开始时建立有界读取边界，
发送确定性的 `nucleiTemplates` manifest 和按 digest 请求的 YAML blob；同一 exchange
内不会重新读取 mutable overlay，也不持久化任务快照或 generation。空集合会映射为稳定的
`no_enabled_templates`，不会回退到禁用、旧 digest 或默认模板。Agent 只处理 Registry
descriptor、完整性、CAS 和固定只读目录，不读取 Nuclei Engine ID 或解析模板业务内容。

manifest/blob 生成阶段会再次校验 canonical `templateId`、安全的 `.yaml`/`.yml`
相对路径、内容 SHA-256、单文档 YAML 和重复 ID；这是持久化数据进入可执行 Engine
之前的最后 Server 边界。Agent 可按 digest 做节点级 CAS 复用；CAS 或发布失败时任务
直接失败且不启动 Nuclei，不提供任务私有 fallback。

终态任务的通知 outbox 写入由 repository 的窄 producer sink 通过同一
GORM 事务完成：`SUCCEEDED` 包含提交 SHA 与导入数量，`FAILED` 只使用已脱敏的
失败代码与摘要。失败文本必须收敛为 Nuclei domain 定义的关闭式代码/摘要组合，
不能把 Git、路径、YAML、诊断或原始错误写入任务通知。Nuclei application ports
不依赖 notification 持久化类型；通知 worker 的后续物化失败也不会改写已提交的
目录或任务事实。

## PostgreSQL advisory-lock 契约验证

该契约测试需要在 Compose 网络内运行，不能在宿主机上用未发布的 PostgreSQL
端口替代。开发 Compose 服务启动后，执行：

```bash
docker exec lunafox-server-1 sh -lc '
  cd /workspace/server &&
  LUNAFOX_NUCLEI_POC_POSTGRES_DSN="host=postgres user=postgres password=postgres dbname=lunafox sslmode=disable search_path=nuclei_poc_activation_contract" \
  /usr/local/go/bin/go test ./internal/modules/nucleipoc/repository \
    -run TestNucleiPOCRepositoryPostgresAdvisoryLockSerializesAcrossConnections \
    -count=1 -v
'
```

测试会在 `nuclei_poc_activation_contract` 独立 schema 中建表并清理，不改动应用
表；输出必须是 `PASS`，而不是因缺少 DSN 的 `SKIP`。
