# Fingerprint Module

该模块只管理一个全局指纹库：`fingerprinthub`。它接收并导出
`fingerprinthub_web.json` JSON aggregate。完整测试语料固定为
0x727/FingerprintHub 提交 `1ca878524ec242a39629a3704532139917779e9c` 的
`web_fingerprint_v4.json`；它是 Observer Ward 默认更新的 Web 制品。

导入在写入前完成 UTF-8、JSON aggregate 和每条 HTTP 规则校验；任意错误都不会
部分写入。`payload JSONB` 是原生规则唯一真源，`name`、`fingerprint_id` 和
`severity` 只是列表查询投影。所有其他库值都会在路由、导入、导出和持久化前失败。

HTTP API 只接受 `/v1/fingerprintLibraries/fingerprinthub`：支持列表、严重度分面、
详情、multipart JSON 导入、批量删除、清空和 `fingerprinthub_web.json` 当前制品导出。
Agent 与 HTTP 下载同一个 current 文件；执行计划只允许
`fingerprintLibraryFingerPrintHub` 平台资源。

`testdata/` 是测试资产，不是运行时种子。`full/fingerprinthub_web.json` 是唯一完整
语料；回归测试校验该文件的 SHA-256、全量数据库导入、原生导出、导出重解析与幂等
再导入。`small/` 保存严格解析的聚焦用例。

安装器的固定发布种子位于仓库根 `resources/fingerprints/web_fingerprint_v4.json`。
它只在 Preclean 后的新安装中、默认字典之后且 Agent 注册之前通过应用层导入；普通
Server 启动、手动 Compose、Clear 和用户删除都不会读取或恢复该文件。它不使用
`testdata/`、语料 manifest 或网络回退。
