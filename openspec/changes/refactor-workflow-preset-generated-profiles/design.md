## Context
当前 workflow 契约为 code-first，schema/docs 已自动生成；但 preset 仍手工维护。  
这会让“可执行契约”与“默认配置”出现长期漂移风险。用户与前端语义上已经将 preset 视为可直接运行模板，因此需要将其治理收敛到同一生成链。

## Goals / Non-Goals
- Goals:
  - 将 workflow `default` 配置收敛到契约源驱动，避免手工复制。
  - 用可选 overlay 表达 `fast/deep` 场景策略，避免三份完整 YAML。
  - 在未上线阶段完成一次性切换，不保留历史 preset 兼容路径。
  - 建立 CI 漂移防护，保证产物可追溯。
- Non-Goals:
  - 不在本次实现运行时动态生成 preset。
  - 不要求每个 workflow 强制提供 `fast/deep` 档位。
  - 不在本次引入外部插件 preset 注册机制。

## Decisions
1. 产物模型固定为“三层”
- 契约层（source of truth）：
  - workflow code-first 契约定义（含字段结构、版本、描述、默认值信息）。
- 策略层（optional）：
  - profile overlay，仅描述差异字段（例如线程、速率、阶段 enable 开关）。
- 产物层：
  - 生成最终 preset YAML，供 server loader 读取。

2. 默认值策略
- `default` preset 必须可由契约层直接生成。
- 若契约缺少某字段默认值且该字段在 schema 下被 `enabled=true` 时 required，生成阶段必须失败，禁止产出不完整默认配置。

3. profile 策略
- `fast/deep` 为可选 profile，不强制全 workflow 覆盖。
- overlay 只允许覆盖存在于契约/schema 的字段。
- overlay 合并后必须通过 schema 校验。

4. 一次性切换策略（pre-launch）
- `server/internal/preset/loader.go` 继续从 `presets/*.yaml` 读取。
- 不保留旧 preset 命名/旧手工文件兼容读取。
- 前后端 preset ID 与展示语义按新生成规则同步更新（允许 breaking）。
- 生成链在构建期运行，运行时只消费静态产物。

5. CI 与治理
- 将 preset 生成纳入 `make generate`（或同等全局入口）。
- CI 必须执行“重新生成后无 diff”检查，覆盖：
  - `server/internal/workflowschema`
  - `docs/config-reference`
  - `server/internal/preset/presets`

## Architecture Sketch
```text
contract_definition (code-first)
  -> workflow-contract-gen
      -> schema (server/internal/workflowschema)
      -> docs (docs/config-reference)
      -> default preset (server/internal/preset/presets/<workflow>.default.yaml)
      -> profile presets (optional, e.g. <workflow>.fast.yaml, <workflow>.deep.yaml)
  -> schema validation gate
  -> preset.Loader embed/load
  -> catalog preset API
```

## Migration Plan
1. 为当前 workflow（`subdomain_discovery`）补齐 default source 与 profile source（可先仅 default）。
2. 在生成器中新增 preset 生成能力并接入现有脚本。
3. 将当前手工 `subdomain_discovery.yaml` 迁移为生成产物（或 alias 到新命名规则）。
4. 清理旧 preset 文件与兼容代码路径，并同步更新前端调用与测试基线。
5. 增加 CI 无差异检查并冻结手工编辑路径。

## Risks / Trade-offs
- 风险：契约未覆盖完整默认值时，生成失败会阻断构建。
  - 缓解：先在单 workflow 上落地并补齐 defaults，再扩展。
- 风险：profile 设计不当可能出现“场景语义漂移”。
  - 缓解：profile 仅允许差异覆盖 + 强制校验 + 人可读注释。
- 风险：迁移期前端依赖 preset id 可能受命名变化影响。
  - 缓解：一次性同步前后端与测试，明确 breaking 改动窗口并快速收敛。

## Alternatives Considered
1. 继续手工维护 preset（维持现状）
- 优点：短期改造最小。
- 缺点：长期漂移风险最高，不满足治理目标。

2. 运行时动态生成 preset
- 优点：减少产物文件。
- 缺点：运行时复杂度和故障面增加，不利于排障和审计。

3. 构建期生成静态 preset（本方案）
- 优点：可审计、可 diff、与现有 loader 兼容。
- 缺点：需要维护生成链与 CI 门禁。
