import { VulnerabilitiesDetailView } from "@/components/vulnerabilities/vulnerabilities-detail-view"

/**
 * Target vulnerabilities page
 * Displays vulnerability details under the target
 */
export default async function TargetVulnerabilitiesPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const targetId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <VulnerabilitiesDetailView targetId={targetId} />
    </div>
  )
}
