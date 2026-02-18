"use client"

import { lazyPage } from "@/components/common/lazy-page"

const SystemLogsView = lazyPage(
  () => import("@/components/settings/system-logs/system-logs-view").then((m) => ({ default: m.SystemLogsView }))
)

export default function SystemLogsPage() {
  return <SystemLogsView />
}
