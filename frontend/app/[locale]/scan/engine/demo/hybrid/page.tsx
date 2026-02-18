"use client"

import { lazyPage } from "@/components/common/lazy-page"

const HybridDemoContent = lazyPage(
  () => import("./content")
)

export default function HybridDemo() {
  return <HybridDemoContent />
}
