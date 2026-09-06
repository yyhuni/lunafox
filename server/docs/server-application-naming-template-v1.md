# Server Application Naming Template v1

本文件定义 `server/internal/modules/*/application` 的默认规则；模块局部 `README.md` 只补充差异化约束和例外，不重复本模板。

## 文档优先级

1. `docs/harness/rules.md`
   仓库级摘要规则，模块文档不得冲突。
2. `server/docs/server-application-naming-template-v1.md`
   server application 默认规则模板。
3. `server/internal/modules/*/application/README.md`
   模块增量规则、例外说明和当前有效示例。

## 默认角色

### Facade

- `*Facade` 表示 provider 模块对外暴露的稳定业务入口。
- handler、gRPC service、其他模块若要消费 provider 模块的业务能力，优先通过 facade 或 facade-backed boundary 暴露。
- facade 负责入口聚合、错误映射、少量日志和边界语义整合。
- facade 中重复出现的边界错误映射、状态映射或入参规整规则，应收口到模块内 `*_facade_helpers.go` 或更明确的 service；单个方法只保留对 helper 的调用。
- facade 不承担复杂 DTO mapper 职责；复杂映射应落到更明确的 mapper 或转换层。
- facade 构造函数只接收已经构造好的 service 或明确边界依赖；内部 query/command/orchestration service 的对象图装配由 `bootstrap/wiring` 作为 composition root 负责，不在 facade constructor 中直接 `New*Service(...)`。

### Orchestration Service

- `*Service` 用于承载 facade 背后的多步骤业务编排。
- 涉及多个 store/port、状态迁移、runtime 解析、阶段推进等流程时，应优先下沉到 service。
- orchestration service 不是 provider 模块的对外总入口；其他模块不应直接依赖它。

### Transport / Adapter Service

- 某些 application `*Service` 可以作为 transport-facing 或 cross-module bridge 型窄职责服务保留。
- 只有在同时满足以下条件时，才应归类为 adapter service：
  - 只服务于单一 transport 或单一桥接链路
  - 只依赖 provider 模块暴露的显式 application port / boundary interface
  - 不聚合 provider 模块的统一业务入口
  - 命名、README 或注释能直接说明其 narrow scope
- consumer 侧允许保留这类 adapter service，但这不改变 provider 模块必须通过 facade 或 facade-backed boundary 对外收口业务能力的要求。

## 命名与边界默认规则

- facade 文件优先使用 `facade_<resource-or-usecase>.go`。
- service、ports、inputs、outputs 文件继续按资源和职责命名，避免新增泛名文件。
- 跨模块 application 依赖面向显式 port 或 boundary interface 建模，不直接依赖对方模块内部 service。
- `bootstrap/wiring` 可以组装 query/command/orchestration service 并注入 facade；不得把这些内部 service 作为 provider 模块的对外业务入口直接暴露，除非模块 README 明确声明其为窄职责 adapter/application service 例外。
- provider 模块是否按单 facade 还是多 facade 拆分，由模块 README 说明。

## 模块 README 必须补充什么

每个 `server/internal/modules/*/application/README.md` 至少说明：

- facade 的拆分维度
- 是否存在允许直接暴露的 transport/adapter service 例外
- 模块特有的命名约束、实现唯一性约束或历史迁移状态

## 示例要求

- README 中列举的 facade/service/ports 示例必须与当前仓库实际文件名一致。
- 如果示例文件被重命名或删除，README 需要同步更新。
