import { TargetOverview } from "@/components/target/target-overview"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

/**
 * Target overview page
 * Displays target statistics and summary information
 */
export default async function TargetOverviewPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const targetId = Number(id)

  return (
    <DetailAssetContentFrame>
      <TargetOverview targetId={targetId} />
    </DetailAssetContentFrame>
  )
}
