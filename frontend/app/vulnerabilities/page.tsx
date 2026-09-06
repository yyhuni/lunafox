import { getTranslations } from "next-intl/server"
import { PageHeader } from "@/components/common/page-header"
import { VulnerabilitiesWorkspace } from "./vulnerabilities-workspace"

/**
 * All vulnerabilities page
 * Displays all vulnerabilities with vertical split layout
 */
export default async function VulnerabilitiesPage() {
  const tVuln = await getTranslations("vulnerabilities")

  return (
    <div className="@container/main flex min-h-0 flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        title={tVuln("title")}
        description={tVuln("description")}
        code="VULN"
      />
      <div className="flex min-h-0 flex-1 flex-col px-4 lg:px-6">
        <VulnerabilitiesWorkspace />
      </div>
    </div>
  )
}
