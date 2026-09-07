import { OverviewPageContent } from "@/components/overview/overview-page-content"

/**
 * Overview page component
 * This is the main overview page of the application, containing cards, charts and data tables
 * Layout structure has been moved to the root layout component
 */
export default function Page() {
  return (
    // Content area containing cards, charts and data tables
    <div className="flex flex-col gap-4 py-4 md:gap-5 md:pt-5 md:pb-6">
      <OverviewPageContent />
    </div>
  )
}
