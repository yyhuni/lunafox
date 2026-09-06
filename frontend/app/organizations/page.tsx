import { PageHeader } from "@/components/common/page-header"
import { OrganizationList } from "@/components/organization/organization-list"
import { getTranslations } from "next-intl/server"

/**
 * Organization management page
 * Sub-page under asset management that displays organization list and related operations
 */
export default async function OrganizationPage() {
  const t = await getTranslations("pages.organization")

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="ORG-01"
        title={t("title")}
        description={t("description")}
      />

      {/* Organization list component */}
      <div className="px-4 lg:px-6">
        <OrganizationList />
      </div>
    </div>
  )
}
