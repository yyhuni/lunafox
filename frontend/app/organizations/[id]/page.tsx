import { OrganizationDetailView } from "@/components/organization/organization-detail-view"

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
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <OrganizationDetailView organizationId={resolvedParams.id} />
    </div>
  )
}
