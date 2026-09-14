import { OrganizationDetailView } from "@/components/organization/organization-detail-view"
import { COMPACT_PAGE_SHELL_CLASS } from "@/components/shared/layout/page-shell-density"

/**
 * Organization detail page
 * Displays organization statistics and asset list
 */
export default async function OrganizationDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const resolvedParams = await params

  return (
    <div className={COMPACT_PAGE_SHELL_CLASS}>
      <OrganizationDetailView organizationId={resolvedParams.id} />
    </div>
  )
}
