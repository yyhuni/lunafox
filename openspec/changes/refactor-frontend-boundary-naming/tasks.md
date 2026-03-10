## 1. OpenSpec and docs
- [x] 1.1 写入 proposal / design / delta spec
- [x] 1.2 写入 `docs/plans/2026-03-06-frontend-boundary-naming-design.md`
- [x] 1.3 写入 `docs/plans/2026-03-06-frontend-boundary-naming.md`
- [x] 1.4 更新 `openspec/project.md` 的命名标准总表
- [x] 1.5 生成 `docs/plans/2026-03-06-frontend-boundary-naming-audit.md`

## 2. Frontend audit
- [x] 2.1 识别候选审计范围中的 A 类文件
- [x] 2.2 识别并记录 B 类文件（含初始 B 类候选）
- [x] 2.3 识别 C 类排除范围
- [x] 2.4 从 A 类清单提取精确实施文件和测试文件

## 3. Frontend boundary cleanup (TDD)
- [x] 3.1 先补分页 / 通知 / service 请求参数的失败测试
- [x] 3.2 仅收敛 A 类 service 请求参数与请求体命名
- [x] 3.3 仅收敛 A 类 DTO、分页解析与通知解析中的双轨兼容字段
- [x] 3.4 删除 A 类中的错误自动转换注释与文档表述
- [x] 3.5 运行 A 类相关前端测试

## 4. Enforcement
- [x] 4.1 扩展 `scripts/ci/check-interface-naming.sh` 的前端规则
- [x] 4.2 将前端规则限制在 A 类审计范围，并显式排除 B 类文件
- [x] 4.3 确保规则只检查代码模式，不因注释或 route 示例文本误报
- [x] 4.4 接入 `.github/workflows/ci.yml`
- [x] 4.5 运行脚本回归测试与真实仓库扫描

## 5. Verification
- [x] 5.1 运行受影响前端测试
- [x] 5.2 运行 `bash scripts/ci/check-interface-naming.sh`
- [x] 5.3 运行 `openspec validate refactor-frontend-boundary-naming --strict --no-interactive`
