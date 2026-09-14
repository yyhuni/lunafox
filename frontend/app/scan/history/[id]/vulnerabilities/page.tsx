import { VulnerabilitiesDetailView } from "@/components/vulnerabilities/vulnerabilities-detail-view"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

export default async function ScanHistoryVulnerabilitiesPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const scanId = Number(id)

  return (
    <DetailAssetContentFrame>
      <VulnerabilitiesDetailView scanId={scanId} />
    </DetailAssetContentFrame>
  )
}
