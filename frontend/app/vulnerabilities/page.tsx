import { getTranslations } from "next-intl/server"
import { PageHeader } from "@/components/common/page-header"
import { VulnerabilitiesWorkspace } from "./vulnerabilities-workspace"
import {
  COMPACT_CONTAINER_PAGE_SHELL_CLASS,
  COMPACT_CONTENT_GUTTER_CLASS,
} from "@/components/shared/layout/page-shell-density"

/**
 * All vulnerabilities page
 * Displays all vulnerabilities with vertical split layout
 */
export default async function VulnerabilitiesPage() {
  const tVuln = await getTranslations("vulnerabilities")

  return (
    <div className={COMPACT_CONTAINER_PAGE_SHELL_CLASS}>
      <PageHeader
        title={tVuln("title")}
        description={tVuln("description")}
        code="VULN"
      />
      <div className={`flex min-h-0 flex-1 flex-col ${COMPACT_CONTENT_GUTTER_CLASS}`}>
        <VulnerabilitiesWorkspace />
      </div>
    </div>
  )
}
