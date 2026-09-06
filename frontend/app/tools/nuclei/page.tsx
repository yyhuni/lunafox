import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/common/page-header"
import NucleiPocCatalogPage from "@/components/tools/nuclei-poc-catalog-page"

export default async function NucleiPage() {
  const t = await getTranslations("pages.nucleiCatalog")

  return (
    <div className="@container/main flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader code="NCL-01" title={t("title")} description={t("description")} />
      <NucleiPocCatalogPage />
    </div>
  )
}
