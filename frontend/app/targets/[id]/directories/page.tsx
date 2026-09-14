import { DirectoriesView } from "@/components/directories/directories-view"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

export default async function TargetDirectoriesPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const targetId = Number(id)

  return (
    <DetailAssetContentFrame>
      <DirectoriesView targetId={targetId} />
    </DetailAssetContentFrame>
  )
}
