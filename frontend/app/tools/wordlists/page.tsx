import { PageHeader } from "@/components/common/page-header"
import WordlistsPageContent from "@/components/tools/wordlists-page"
import {
  WORDLISTS_CONTENT_SHELL_CLASS,
  WORDLISTS_PAGE_SHELL_CLASS,
} from "@/components/tools/wordlists-page-layout"
import { getTranslations } from "next-intl/server"

export default async function WordlistsPage() {
  const t = await getTranslations("pages.tools.wordlists")

  return (
    <div className={WORDLISTS_PAGE_SHELL_CLASS}>
      <PageHeader
        code="WDL-01"
        title={t("title")}
        description={t("description")}
      />

      <div className={WORDLISTS_CONTENT_SHELL_CLASS}>
        <WordlistsPageContent />
      </div>
    </div>
  )
}
