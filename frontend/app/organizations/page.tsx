import { PageHeader } from "@/components/common/page-header"
import { InlineHelpTooltip } from "@/components/common/inline-help-tooltip"
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
  const tTooltips = await getTranslations("tooltips")

  return (
    <div className={COMPACT_PAGE_SHELL_CLASS}>
      <PageHeader
        code="ORG-01"
        title={t("title")}
        description={t("description")}
        descriptionSupplement={(
          <InlineHelpTooltip ariaLabel={t("title")}>
            {tTooltips("organizationConcept")}
          </InlineHelpTooltip>
        )}
      />

      {/* Organization list component */}
      <div className={COMPACT_CONTENT_GUTTER_CLASS}>
        <OrganizationList />
      </div>
    </div>
  )
}
