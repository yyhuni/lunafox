import { OverviewPageContent } from "@/components/overview/overview-page-content"
import { COMPACT_PAGE_RHYTHM_CLASS } from "@/components/shared/layout/page-shell-density"

/**
 * Overview page component
 * This is the main overview page of the application, containing cards, charts and data tables
 * Layout structure has been moved to the root layout component
 */
export default function Page() {
  return (
    // Content area containing cards, charts and data tables
    <div className={`flex flex-col ${COMPACT_PAGE_RHYTHM_CLASS}`}>
      <OverviewPageContent />
    </div>
  )
}
