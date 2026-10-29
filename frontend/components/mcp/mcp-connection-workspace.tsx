"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { PageHeader } from "@/components/common/page-header"
import { COMPACT_CONTENT_GUTTER_CLASS, COMPACT_FLEX_PAGE_SHELL_CLASS } from "@/components/shared/layout/page-shell-density"
import { cn } from "@/lib/utils"
import { McpConnectionSettings } from "./mcp-connection-settings"
import { McpConnectionFlow } from "./mcp-connection-flow"
import type { McpClient } from "./mcp-client-configuration"

export function McpConnectionWorkspace() {
  const t = useTranslations("mcpIntegration")
  const [client, setClient] = useState<McpClient>("claudeCode")

  return (
    <div className={COMPACT_FLEX_PAGE_SHELL_CLASS}>
      <PageHeader title={t("title")} description={t("description")} code="MCP" />
      <div className={cn(COMPACT_CONTENT_GUTTER_CLASS, "space-y-3")}>
        <McpConnectionFlow client={client} />
        <McpConnectionSettings client={client} onClientChange={setClient} />
      </div>
    </div>
  )
}
