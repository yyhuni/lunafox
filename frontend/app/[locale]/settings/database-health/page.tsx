"use client"

import { lazyPage } from "@/components/common/lazy-page"

const DatabaseHealthView = lazyPage(
  () => import("@/components/settings/database-health/database-health-view").then((m) => ({ default: m.DatabaseHealthView }))
)

export default function DatabaseHealthPage() {
  return <DatabaseHealthView />
}
