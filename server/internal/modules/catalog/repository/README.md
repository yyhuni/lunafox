# catalog/repository

catalog 模块 repository 规范：

- `target.go`：`TargetRepository` 结构体与构造函数。
- `target_query.go` / `target_command.go`：Target 查询/写操作。
- `target_mapper.go`：Target 的 `model <-> domain` 映射。
- `target_stats_query.go`：Target 统计查询（资产与漏洞计数）。
- `engine.go` + `engine_query.go` + `engine_command.go`：Engine 三分结构。
- `engine_mapper.go`：Engine 的 `model <-> domain` 映射。
- `wordlist.go` + `wordlist_query.go` + `wordlist_command.go`：Wordlist 三分结构。
- `wordlist_mapper.go`：Wordlist 的 `model <-> domain` 映射。
- `subfinder_provider_settings.go` + `*_query.go` + `*_command.go`：Provider 配置三分结构。
- `subfinder_provider_settings_mapper.go`：Provider Settings 的 `model <-> domain` 映射。

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。
