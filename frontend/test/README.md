# Frontend Quality Gates

在 `/Users/yangyang/Desktop/lunafox/frontend` 目录执行以下命令：

```bash
pnpm run typecheck
pnpm run lint:core
pnpm run check:datatable-legacy
pnpm run test
pnpm run test:coverage
```

说明：

- `typecheck`: TypeScript 无输出编译检查
- `lint:core`: 核心业务目录 ESLint 门禁
- `check:datatable-legacy`: 防止 `UnifiedDataTable` legacy props 回流
- `test`: Vitest 非 watch 模式（CI 友好）
- `test:coverage`: 生成覆盖率报告
