import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/common/page-header"
import NucleiPocCatalogPage from "@/components/tools/nuclei-poc-catalog-page"
import { COMPACT_CONTAINER_PAGE_SHELL_CLASS } from "@/components/shared/layout/page-shell-density"

export default async function NucleiPage() {
  const t = await getTranslations("pages.nucleiCatalog")

  return (
    <div className={COMPACT_CONTAINER_PAGE_SHELL_CLASS}>
      <PageHeader code="NCL-01" title={t("title")} description={t("description")} />
      <NucleiPocCatalogPage />
    </div>
  )
}
