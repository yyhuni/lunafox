import { DirectoriesView } from "@/components/directories/directories-view"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

export default async function ScanDirectoriesPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const scanId = Number(id)

  return (
    <DetailAssetContentFrame>
      <DirectoriesView scanId={scanId} />
    </DetailAssetContentFrame>
  )
}
