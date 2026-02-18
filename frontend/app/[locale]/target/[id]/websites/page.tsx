"use client"

import { lazyPage } from "@/components/common/lazy-page"
import { useParams } from "next/navigation"

const WebSitesView = lazyPage(
  () => import("@/components/websites/websites-view").then((m) => ({ default: m.WebSitesView }))
)

export default function WebSitesPage() {
  const { id } = useParams<{ id: string }>()
  const targetId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <WebSitesView targetId={targetId} />
    </div>
  )
}
