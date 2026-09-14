import { ScreenshotsGallery } from "@/components/screenshots/screenshots-gallery"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

export default async function ScreenshotsPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const targetId = Number(id)

  return (
    <DetailAssetContentFrame>
      <ScreenshotsGallery targetId={targetId} />
    </DetailAssetContentFrame>
  )
}
