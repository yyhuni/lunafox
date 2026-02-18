# Frontend 巡检失败记录

> 记录每个失败项的复现路径、错误信息、修复状态，便于长任务续跑。

| 时间 | 路由 | 场景 | 现象 | 控制台/报错 | 修复状态 | 关联文件 |
|---|---|---|---|---|---|---|
| 待执行 | - | - | - | - | - | - |

## 备注
- 控制台错误优先记录：`Hydration failed`、`Recoverable Error`、`Unhandled`。
- 对于不可复现问题，标记为 `needs-info` 并注明环境条件。

| 2026-02-13T04:14:53.032Z | /zh/dashboard | locale=zh | quick-scan dialog did not open | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/settings/system-logs | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/settings/notifications | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/settings/workers | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1 | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1/details | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1/directories | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1/endpoints | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1/ip-addresses | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1/overview | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1/screenshots | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1/settings | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1/subdomain | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/target/1/websites | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:14:53.032Z | /zh/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |

| 2026-02-13T04:21:58.712Z | /zh/settings/workers | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/dashboard | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1 | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1/directories | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1/endpoints | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1/ip-addresses | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1/details | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1/subdomain | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1/screenshots | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1/settings | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1/overview | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1/websites | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:21:58.712Z | /zh/target/1/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |

| 2026-02-13T04:28:22.216Z | /zh/dashboard | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/settings/workers | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target/1/directories | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target/1/endpoints | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target/1/details | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target/1/overview | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target/1/ip-addresses | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target/1/screenshots | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target/1/settings | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target/1/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target/1/subdomain | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/target/1/websites | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:22.216Z | /zh/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |

| 2026-02-13T04:28:51.013Z | /zh/dashboard | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target/1/details | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/settings/workers | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target/1/directories | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target/1/endpoints | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target/1/ip-addresses | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target/1/overview | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target/1/screenshots | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target/1/settings | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target/1/subdomain | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target/1/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/target/1/websites | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:28:51.013Z | /zh/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |

| 2026-02-13T04:34:50.239Z | /zh/dashboard | locale=zh | missing quick-scan button | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |

| 2026-02-13T04:39:22.293Z | /zh/dashboard | locale=zh | missing quick-scan button | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/search | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/settings/blacklist | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/scan/scheduled | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/settings/database-health | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/settings/workers | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/settings/system-logs | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/settings/notifications | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1/details | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1/directories | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1/endpoints | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1 | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1/settings | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1/overview | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1/screenshots | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1/ip-addresses | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1/subdomain | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:39:22.293Z | /zh/target/1/websites | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |

| 2026-02-13T04:41:29.317Z | /zh/settings/blacklist | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/scan/scheduled | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/dashboard | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/settings/database-health | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/settings/system-logs | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/settings/workers | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/settings/notifications | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1/details | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1/endpoints | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1 | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1/directories | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1/overview | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1/screenshots | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1/ip-addresses | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1/settings | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1/websites | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1/subdomain | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:41:29.317Z | /zh/target/1/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |

| 2026-02-13T04:42:36.590Z | /zh/dashboard | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/scan/scheduled | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/settings/blacklist | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/settings/database-health | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/settings/notifications | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/settings/system-logs | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/settings/workers | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1 | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1/details | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1/directories | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1/endpoints | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1/ip-addresses | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1/overview | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1/screenshots | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1/settings | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1/subdomain | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/target/1/websites | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
| 2026-02-13T04:42:36.590Z | /zh/vulnerabilities | locale=zh | http-500 | - | pending-fix | frontend/components/auth/auth-layout.tsx, frontend/components/scan/quick-scan-dialog.tsx |
