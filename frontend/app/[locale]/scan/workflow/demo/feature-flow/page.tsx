"use client"

import { lazyPage } from "@/components/common/lazy-page"

const FeatureFlowDemoContent = lazyPage(
  () => import("./content")
)

export default function FeatureFlowDemoPage() {
  return <FeatureFlowDemoContent />
}
