import { EndpointsDetailView } from "@/components/endpoints/endpoints-detail-view"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"
/**
 * Target endpoints page
 * Displays endpoint details under the target
 */
export default async function TargetEndpointsPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params

  return (
    <DetailAssetContentFrame>
      <EndpointsDetailView targetId={Number(id)} />
    </DetailAssetContentFrame>
  )
}
