# test-plan 使用说明

## 文件说明
- `routes.todo.json`：路由级任务清单（可机器更新）。
- `progress.md`：轮次级汇总（给人看）。
- `failures.md`：失败明细与修复跟踪。

## 生成/刷新路由清单
```bash
cd /Users/yangyang/Desktop/lunafox/frontend
node scripts/build-route-todo.mjs
```

## 默认排除规则
- `/[locale]/tools/**`
- `/[locale]/prototypes/**`
- 任意包含 `/demo/` 的路由

## 建议执行节奏
1. 并发跑 smoke，更新 `routes.todo.json` 状态。
2. 有失败就写入 `failures.md`，并修复代码。
3. 修复后补回归测试，再跑受影响路由。
4. 全部通过后更新 `progress.md`。
