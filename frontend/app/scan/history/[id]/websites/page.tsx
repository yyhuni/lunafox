import { WebSitesView } from "@/components/websites/websites-view"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

export default async function ScanWebSitesPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const scanId = Number(id)

  return (
    <DetailAssetContentFrame>
      <WebSitesView scanId={scanId} />
    </DetailAssetContentFrame>
  )
}
