import { EndpointsDetailView } from "@/components/endpoints/endpoints-detail-view"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

export default async function ScanHistoryEndpointsPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params

  return (
    <DetailAssetContentFrame>
      <EndpointsDetailView scanId={Number(id)} />
    </DetailAssetContentFrame>
  )
}
