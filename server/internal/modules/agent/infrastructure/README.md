# agent/infrastructure

agent 模块 infrastructure 规范：

- `clock.go`：`Clock` 端口默认实现。
- `token_generator.go`：`TokenGenerator` 端口默认实现。
- 默认实现仅负责外部能力封装（如时间、随机数），不承载业务流程。

约束：

- application 只定义端口，不直接持有外部能力实现。
- 具体实现通过 `bootstrap/wiring` 注入 application service/facade。
