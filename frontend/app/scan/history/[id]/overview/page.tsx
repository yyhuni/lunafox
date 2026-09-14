import { ScanOverview } from "@/components/scan/history/scan-overview"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

/**
 * Scan overview page
 * Displays scan statistics and summary information
 */
export default async function ScanOverviewPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const scanId = Number(id)

  return (
    <DetailAssetContentFrame className="flex min-h-0 min-w-0 flex-1 flex-col">
      <ScanOverview scanId={scanId} />
    </DetailAssetContentFrame>
  )
}
