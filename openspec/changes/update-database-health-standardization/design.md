## Context
数据库健康页是运维导向页面，需要反映数据库真实可用性与风险水平。当前前端已经有展示组件与轮询骨架，但缺少后端接口与统一判定来源，导致：
- 前端阈值和状态逻辑难以审计与复用。
- `offline` 与“性能变差”的语义混用。
- 时间字段不标准，影响时区一致性和自动化处理。

本次变更是跨前后端的行为收敛，不是单纯页面美化。

## Goals / Non-Goals
- Goals:
  - 提供生产可用的数据库健康快照 API。
  - 用后端统一输出健康状态，前端不再自行定义核心状态语义。
  - 固化核心指标最小集，降低实现复杂度并提高可靠性。
  - 标准化时间字段与错误/陈旧数据反馈。
- Non-Goals:
  - 不在本次引入全量数据库性能分析平台能力（如完整慢查询分析、容量预测仪表盘）。
  - 不在本次引入复杂流式推送（SSE/WebSocket）；仍使用轮询。

## Decisions
- Decision: 健康状态由后端计算并直接返回。
  - Rationale: 避免多端阈值漂移与语义不一致，便于审计和统一告警策略。
  - Alternatives considered:
    - 前端继续硬编码阈值: 实现快但长期不可维护。
    - 双端各自计算: 易出现冲突，不可取。

- Decision: 区分核心指标与可选指标。
  - Core metrics (must-have):
    - 探活可用性（probe success/latency）
    - 连接利用率（used/max）
    - 复制延迟（适用时）
    - 备份新鲜度
  - Optional metrics (best-effort):
    - qps
    - walGeneratedMb24h
    - cacheHitRate
  - Rationale: 核心指标用于健康判定；可选指标用于观察趋势，不应单独触发 `offline`。

- Decision: `offline` 仅表示不可用或探活失败，不表示“阈值超限”。
  - `offline`: 数据库不可连接、探活超时、关键依赖失败。
  - `degraded`: 可用但核心指标超阈值，或仅部分信号不可用。
  - `maintenance`: 明确维护模式（后端显式设定）。
  - `online`: 核心检查通过且未触发降级。

- Decision: 时间字段统一为 ISO 8601。
  - API 字段使用绝对时间（例如 `observedAt`, `lastCheckAt`, `alerts[].occurredAt`）。
  - 前端负责显示本地化相对时间（如“3m ago”）。

- Decision: 支持部分失败降级。
  - 当可选指标采集失败时，接口仍返回可用数据，并在响应中标记缺失项。
  - 避免因单一统计查询失败导致页面整体不可用。

## API Shape (target)
- Top-level:
  - `status`
  - `observedAt` (ISO 8601)
  - `role`, `readOnly`, `version`
  - `uptimeSeconds`
  - `coreSignals`
  - `optionalSignals`
  - `alerts[]`
  - `unavailableSignals[]`
- Compatibility:
  - 若前端改造分阶段进行，后端可短期提供兼容字段映射，最终收敛到标准字段。

## Risks / Trade-offs
- 风险: 不同 PostgreSQL 部署形态下部分统计视图权限不足。
  - Mitigation: 为受限指标提供 `unavailableSignals` 标记与降级策略。

- 风险: 阈值默认值不适配所有环境。
  - Mitigation: 后端集中配置阈值并提供合理默认值，后续按环境调整。

- 风险: 前端切换字段后与 mock 数据不一致。
  - Mitigation: 同步更新 mock 数据模型，并增加 hook/service 的契约测试。

## Migration Plan
1. 后端新增 `/api/system/database-health/`，先提供稳定核心字段。
2. 前端切换为后端状态驱动，移除前端核心状态推导。
3. 新增或更新测试覆盖关键路径。
4. 确认 mock 与 real API 结构一致，避免环境切换问题。

## Open Questions
- `maintenance` 状态的来源是否由配置开关控制，还是由数据库只读/管理窗口自动推断。
- 是否需要在第一阶段就暴露阈值配置到管理页（当前建议不做）。
