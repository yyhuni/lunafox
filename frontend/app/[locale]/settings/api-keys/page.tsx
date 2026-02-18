"use client"

import { lazyPage } from "@/components/common/lazy-page"

const ApiKeysSettingsPageContent = lazyPage(
  () => import("./content")
)

export default function ApiKeysSettingsPage() {
  return <ApiKeysSettingsPageContent />
}
