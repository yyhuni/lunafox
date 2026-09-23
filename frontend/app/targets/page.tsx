import { PageHeader } from "@/components/common/page-header"
import { InlineHelpTooltip } from "@/components/common/inline-help-tooltip"
import { AllTargetsDetailView } from "@/components/target/all-targets-detail-view"
import { getTranslations } from "next-intl/server"
import {
  COMPACT_CONTENT_GUTTER_CLASS,
  COMPACT_PAGE_SHELL_CLASS,
} from "@/components/shared/layout/page-shell-density"

export default async function AllTargetsPage() {
  const t = await getTranslations("pages.target")
  const tTooltips = await getTranslations("tooltips")

  return (
    <div className={COMPACT_PAGE_SHELL_CLASS}>
      <PageHeader
        code="TGT-01"
        title={t("title")}
        description={t("description")}
        descriptionSupplement={(
          <InlineHelpTooltip ariaLabel={t("title")}>
            {tTooltips("targetConcept")}
          </InlineHelpTooltip>
        )}
      />

      {/* Target list */}
      <div className={COMPACT_CONTENT_GUTTER_CLASS}>
        <AllTargetsDetailView />
      </div>
    </div>
  )
}
