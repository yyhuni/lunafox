"use client"

import { lazyPage } from "@/components/common/lazy-page"
import { useParams } from "next/navigation"

const SubdomainsDetailView = lazyPage(
  () => import("@/components/subdomains/subdomains-detail-view").then((m) => ({ default: m.SubdomainsDetailView }))
)

export default function TargetSubdomainPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="px-4 lg:px-6">
      <SubdomainsDetailView targetId={Number(id)} />
    </div>
  )
}
