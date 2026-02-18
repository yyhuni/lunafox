"use client"

import { lazyPage } from "@/components/common/lazy-page"
import { useParams } from "next/navigation"

const EndpointsDetailView = lazyPage(
  () => import("@/components/endpoints/endpoints-detail-view").then((m) => ({ default: m.EndpointsDetailView }))
)

export default function ScanHistoryEndpointsPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="px-4 lg:px-6">
      <EndpointsDetailView scanId={Number(id)} />
    </div>
  )
}
