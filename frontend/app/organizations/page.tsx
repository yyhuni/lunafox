import { PageHeader } from "@/components/common/page-header"
import { OrganizationList } from "@/components/organization/organization-list"
import { getTranslations } from "next-intl/server"
import {
  COMPACT_CONTENT_GUTTER_CLASS,
  COMPACT_PAGE_SHELL_CLASS,
} from "@/components/shared/layout/page-shell-density"

/**
 * Organization management page
 * Sub-page under asset management that displays organization list and related operations
 */
export default async function OrganizationPage() {
  const t = await getTranslations("pages.organization")

  return (
    <div className={COMPACT_PAGE_SHELL_CLASS}>
      <PageHeader
        code="ORG-01"
        title={t("title")}
        description={t("description")}
      />

      {/* Organization list component */}
      <div className={COMPACT_CONTENT_GUTTER_CLASS}>
        <OrganizationList />
      </div>
    </div>
  )
}
