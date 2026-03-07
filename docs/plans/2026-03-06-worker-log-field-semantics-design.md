# Worker Log Field Semantics Design

## Goal
把仓库里剩余的 worker 应用日志字段收口到统一的语义化点分命名，并把这条规则纳入 CI，避免再回流 `camelCase` zap key。

## Chosen approach
采用 focused cleanup：
1. 只改剩余 worker/runtime 日志字段
2. 不碰 HTTP OTel 语义字段
3. 不碰 Loki / Prometheus labels
4. 用测试 + CI 脚本双重锁定新规范

## Key decisions
- 应用日志字段继续使用 semantic dotted fields，而不是为 Loki 采集改成 `snake_case`
- `camelCase` worker log keys 一次性直接替换，不保留 alias
- CI 只拦截 worker/server 非测试源码中的 ad-hoc `camelCase` zap key；OTel 语义字段走 allowlist

## Planned verification
- 定点 worker 日志测试
- `scripts/ci/check-interface-naming-test.sh`
- `scripts/ci/check-interface-naming.sh`
- `openspec validate refactor-worker-log-field-semantics --strict --no-interactive`
