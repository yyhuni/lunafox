# Interface Naming Enforcement Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为边界命名标准补上自动检查脚本和 GitHub Actions 门禁，防止 `snake_case` 契约与裸 context key 回流。

**Architecture:** 新增一个仓库级 shell 检查脚本，基于 `rg` 对少量高价值规则做静态扫描，并通过明确 allowlist 避免误报。CI 继续复用现有 `ci.yml` 的路径过滤，在相关目录变更时运行该脚本。

**Tech Stack:** Bash, ripgrep, GitHub Actions, OpenSpec

---

### Task 1: 同步设计与 OpenSpec

**Files:**
- Create: `docs/plans/2026-03-06-interface-naming-enforcement-design.md`
- Create: `docs/plans/2026-03-06-interface-naming-enforcement.md`
- Modify: `openspec/changes/refactor-interface-naming-standards/proposal.md`
- Modify: `openspec/changes/refactor-interface-naming-standards/design.md`
- Modify: `openspec/changes/refactor-interface-naming-standards/tasks.md`

**Step 1: 写入设计文档**

将脚本入口、检查范围、allowlist 和 CI 接入点写入设计文档。

**Step 2: 更新 OpenSpec 变更说明**

把“自动化守门”补入 proposal / design / tasks，确保规范与实现一致。

**Step 3: 运行 OpenSpec 校验**

Run: `openspec validate refactor-interface-naming-standards --strict --no-interactive`
Expected: PASS

### Task 2: 先写失败的脚本回归测试

**Files:**
- Create: `scripts/ci/check-interface-naming-test.sh`

**Step 1: 写测试夹具**

用临时目录生成 bad / good fixture，覆盖：`snake_case` JSON tag、裸 `zap` 字段、裸 Gin context key、错误字段、snake_case path param。

**Step 2: 先运行测试并确认失败**

Run: `bash scripts/ci/check-interface-naming-test.sh`
Expected: FAIL，因为 `scripts/ci/check-interface-naming.sh` 尚未实现或规则未齐全。

### Task 3: 实现检查脚本

**Files:**
- Create: `scripts/ci/check-interface-naming.sh`

**Step 1: 写最小实现**

实现仓库根目录定位、规则执行、allowlist、统一失败输出。

**Step 2: 运行脚本测试直到通过**

Run: `bash scripts/ci/check-interface-naming-test.sh`
Expected: PASS

**Step 3: 运行真实仓库扫描**

Run: `bash scripts/ci/check-interface-naming.sh`
Expected: PASS

### Task 4: 接入 GitHub Actions

**Files:**
- Modify: `.github/workflows/ci.yml`

**Step 1: 增加路径过滤输出**

为接口命名检查增加 `interface_naming` 过滤器和 `run_interface_naming` 输出。

**Step 2: 增加独立 job**

新增 job 执行 `bash scripts/ci/check-interface-naming.sh`。

**Step 3: 做一次 YAML 级静态检查**

Run: `ruby -e 'require "yaml"; YAML.load_file(".github/workflows/ci.yml"); puts "ok"'`
Expected: `ok`

### Task 5: 收尾验证

**Files:**
- Modify: `openspec/changes/refactor-interface-naming-standards/tasks.md`

**Step 1: 标记任务完成**

把新增任务更新为已完成。

**Step 2: 运行最终校验**

Run: `bash scripts/ci/check-interface-naming-test.sh`
Run: `bash scripts/ci/check-interface-naming.sh`
Run: `openspec validate refactor-interface-naming-standards --strict --no-interactive`
Expected: 全部通过
