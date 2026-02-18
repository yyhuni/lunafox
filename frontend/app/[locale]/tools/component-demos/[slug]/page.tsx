"use client"

import { lazyPage } from "@/components/common/lazy-page"

const ComponentDemoPageContent = lazyPage(
  () => import("./content")
)

export default function ComponentDemoPage() {
  return <ComponentDemoPageContent />
}
