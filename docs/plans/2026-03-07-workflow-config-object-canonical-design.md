# Workflow Config Object Canonical Design

## Objective

把 workflow 配置从“字符串驱动”改成“对象驱动”，并确保：

- profile 文件不再出现 YAML-in-YAML；
- server / frontend / persistence 共享同一份 canonical object；
- YAML 退回到编辑器、文件语法与必要边界投影角色；
- defaulting、overlay merge、schema validation 都在对象层完成。

## Why The Current Shape Is Fragile

当前系统的最大问题不是 YAML，而是 **配置在系统里被当成字符串传递**：

- profile 文件外层是 YAML，`configuration` 内层又是一段 YAML 字符串；
- catalog 与 scan API 暴露字符串配置；
- server 在 scan create 时再把字符串 parse 成对象；
- frontend 通过字符串拼接和正则去做配置合并与 capability 判断；
- DB 把 canonical 配置持久化为文本。

这会导致：

1. 同一配置在多个边界重复 parse / marshal；
2. merge/defaulting/validation 没有稳定对象层入口；
3. UI 无法安全地对配置做局部操作；
4. profile 生成链已经 modernized，但最终产物仍保留历史字符串结构。

## End-State Model

推荐终局模型：

- **canonical 形态：对象**
- **文件语法：YAML**
- **API 传输：JSON object**
- **持久化：JSONB**
- **编辑器视图：YAML 文本（对象的投影）**

也就是说，YAML 仍保留，但不再承担系统内主状态职责。

## Canonical Data Model

推荐统一引入一个语义名，例如：

```go
type WorkflowConfig map[string]any
```

顶层对象仍然保持：

```yaml
subdomain_discovery:
  recon:
    enabled: true
```

区别只在于：

- profile loader 直接得到对象；
- scan create 直接接收对象；
- persistence 直接存对象；
- runtime 若还要 YAML，则在最外层 adapter 再转文本。

## Profile Artifact Shape

新的 profile 形态应当是：

```yaml
id: subdomain_discovery.default
name: 子域名发现
description: 生成默认预设
configuration:
  subdomain_discovery:
    recon:
      enabled: true
      tools:
        subfinder:
          enabled: true
          timeout-runtime: 3600
          threads-cli: 10
```

而不是：

```yaml
configuration: |
  subdomain_discovery:
    ...
```

前者只有一个 YAML 文档；后者其实是“外层 YAML + 内层 YAML 字符串”。

## Server Pipeline

server 侧推荐顺序：

1. 从 profile 文件或 API 请求获得 object root
2. 应用 overlay / defaulting
3. 执行 schema validation
4. 持久化 canonical object 到 `scan.configuration`
5. 派生 workflow slice object 到 `scan_task.workflow_config`
6. 如果 runtime 边界仍需要 YAML，再在 outbound mapper 生成 `workflowConfigYAML`

这样做的好处：

- DB 存的是最终执行对象，而不是某次文本序列化结果；
- retry / audit / requeue 面对的是同一个 canonical 状态；
- runtime YAML 只是一份派生视图，不会反向污染事实源。

## Frontend Pipeline

前端推荐顺序：

1. hooks / services 收到对象配置
2. dialog / selector / editor state 持有对象配置
3. capability 解析、merge、preset 应用都在对象层完成
4. 文本编辑器打开时，把对象序列化成 YAML
5. 用户保存时，再把 YAML parse 回对象

这样可以去掉：

- 文本拼接式 merge
- 正则判断 capability
- “切换 preset 就覆盖整段 YAML 字符串”的脆弱逻辑

## Migration Boundary

这次是 pre-launch 改造，可以一次性 cutover，但实现上仍建议分层推进：

1. 先改 profile + catalog
2. 再改 scan create + persistence
3. 再改 frontend state
4. 最后清理 legacy YAML 字段和旧文档

如果 runtime protobuf 这轮不一起动，也没关系；只要明确它是 **adapter boundary**，不是 canonical store。

## Testing Strategy

测试重点不是“YAML 文本长什么样”，而是“对象语义是不是稳定”：

1. profile loader / validator 测试：锁定 mapping 形态
2. catalog handler / DTO 测试：锁定 object API
3. scan create / persistence 测试：锁定 JSONB canonical
4. runtime mapper 测试：锁定 YAML 只在 outbound boundary 派生
5. frontend helper / state 测试：锁定对象 merge 和对象 ⇄ YAML 投影

所有实现都应按 TDD 推进：先写失败测试，再写最小实现。

## Final Recommendation

这次改造的目标不是“抛弃 YAML”，而是：

- **对象是事实源**
- **YAML 是人类视图**

只要这条边界立住，workflow profile、preset 生成、scan create、持久化与 UI 编辑器的复杂度都会明显下降。
