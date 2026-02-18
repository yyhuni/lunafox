"use client"

import { lazyPage } from "@/components/common/lazy-page"

const ComponentDemosIndexPageContent = lazyPage(
  () => import("./content")
)

export default function ComponentDemosIndexPage() {
  return <ComponentDemosIndexPageContent />
}
