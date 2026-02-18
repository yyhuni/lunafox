"use client"

import { lazyPage } from "@/components/common/lazy-page"
import { useParams } from "next/navigation"

const IPAddressesView = lazyPage(
  () => import("@/components/ip-addresses/ip-addresses-view").then((m) => ({ default: m.IPAddressesView }))
)

export default function TargetIPsPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="px-4 lg:px-6">
      <IPAddressesView targetId={Number(id)} />
    </div>
  )
}
