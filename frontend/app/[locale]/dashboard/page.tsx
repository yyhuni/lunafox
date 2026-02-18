import { BauhausDashboardHeader } from "@/components/dashboard/bauhaus-dashboard-header"
import { DashboardLazySections } from "@/components/dashboard/dashboard-lazy-sections"

/**
 * Dashboard page component
 * This is the main dashboard page of the application, containing cards, charts and data tables
 * Layout structure has been moved to the root layout component
 */
export default function Page() {
  return (
    // Content area containing cards, charts and data tables
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      {/* Bauhaus-style dashboard header, shown only in the Bauhaus theme */}
      <BauhausDashboardHeader />

      <DashboardLazySections />
    </div>
  )
}
