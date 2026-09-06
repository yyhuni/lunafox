# server/scripts

按改动范围选择需要的守卫/检查脚本。所有脚本在仓库根运行（`cd /path/to/repo`）。

## 通用原则
- 带 `*-selftest.sh` 的脚本是对应守卫的自测。
- 出现失败请先读脚本输出，通常给出具体文件/规则提示。
- 需要 bash、go、git 可用（部分脚本仅使用 grep/find，不依赖 Go）。
- 跨边界契约漂移统一走仓库根 `make verify-boundary-contracts`；server 局部脚本不要各自维护 AIP legacy 例外 allowlist。

## 守卫与检查
- `check-router-boundaries.sh` / `check-router-boundaries-selftest.sh`
  - 适用：改动路由分发/注册逻辑时运行。
- `check-handler-boundaries.sh` / `check-handler-boundaries-selftest.sh`
  - 适用：改动 handler 时运行。
  - 额外约束：拦截 handler 直接依赖 concrete application `*Service` 或 domain repository 接口的情况，要求改为 facade 或显式 application 接口。
- `check-repository-boundaries.sh` / `check-repository-boundaries-selftest.sh`
  - 适用：改动 repository 时运行。
- `check-dto-boundaries.sh` / `check-dto-boundaries-selftest.sh`
  - 适用：改动 DTO 边界时运行。
- `check-layer-dependencies.sh` / `check-layer-dependencies-selftest.sh`
  - 适用：分层依赖守卫。
- `check-domain-purity.sh`
  - 适用：领域层纯度守卫。
- `check-naming-conventions.sh`
  - 适用：命名规范检查。
- `check-mapper-naming.sh`
  - 适用：mapper 命名检查。
- `check-wiring-conventions.sh` / `check-wiring-conventions-selftest.sh`
  - 适用：改动 `server/internal/bootstrap/wiring/**` 时必跑，检查 wiring 命名和适配器导出约束。
  - 额外约束：允许 `exports.go` 作为 composition root 组装内部 service 后注入 facade，但拦截直接 return 模块内部 `New*Service(...)` 的情况。
- `check-application-composition.sh`
  - 适用：改动 `server/internal/modules/*/application/facade*.go` 时运行。
  - 额外约束：facade / application-boundary constructor 不直接 `New*Service(...)`，子 service 由 bootstrap/wiring composition root 组装后注入。
- `check-no-module-httpdto-adapters.sh`
  - 适用：通用 HTTP DTO 收口检查，阻止重新引入模块内 `dto/common_http.go` 适配层。
- `check-no-legacy-layer.sh`
  - 适用：阻止 legacy 层残留。

## Codex 覆盖循环（仓库根 scripts/codex/coverage-loop/）
- 参见 `../../scripts/codex/coverage-loop/README.md`。
