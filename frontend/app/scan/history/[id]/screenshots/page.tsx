import { ScreenshotsGallery } from "@/components/screenshots/screenshots-gallery"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

export default async function ScanScreenshotsPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const scanId = Number(id)

  return (
    <DetailAssetContentFrame>
      <ScreenshotsGallery scanId={scanId} />
    </DetailAssetContentFrame>
  )
}
