"use client"

import { lazyPage } from "@/components/common/lazy-page"
import { useParams } from "next/navigation"

const EndpointsDetailView = lazyPage(
  () => import("@/components/endpoints/endpoints-detail-view").then((m) => ({ default: m.EndpointsDetailView }))
)
/**
 * Target endpoints page
 * Displays endpoint details under the target
 */
export default function TargetEndpointsPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="px-4 lg:px-6">
      <EndpointsDetailView targetId={Number(id)} />
    </div>
  )
}
