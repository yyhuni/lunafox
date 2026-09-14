import { VulnerabilitiesDetailView } from "@/components/vulnerabilities/vulnerabilities-detail-view"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

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
    <DetailAssetContentFrame>
      <VulnerabilitiesDetailView targetId={targetId} />
    </DetailAssetContentFrame>
  )
}
