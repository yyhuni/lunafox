import { PageHeader } from "@/components/common/page-header"
import { AllTargetsDetailView } from "@/components/target/all-targets-detail-view"
import { getTranslations } from "next-intl/server"
import {
  COMPACT_CONTENT_GUTTER_CLASS,
  COMPACT_PAGE_SHELL_CLASS,
} from "@/components/shared/layout/page-shell-density"

export default async function AllTargetsPage() {
  const t = await getTranslations("pages.target")

  return (
    <div className={COMPACT_PAGE_SHELL_CLASS}>
      <PageHeader
        code="TGT-01"
        title={t("title")}
        description={t("description")}
      />

      {/* Target list */}
      <div className={COMPACT_CONTENT_GUTTER_CLASS}>
        <AllTargetsDetailView />
      </div>
    </div>
  )
}
