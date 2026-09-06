import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"
import { TargetWebsiteEvidenceView } from "@/components/websites/website-relation-evidence-view"

export default async function WebSitesPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const targetId = Number(id)

  return (
    <DetailAssetContentFrame>
      <TargetWebsiteEvidenceView targetId={targetId} />
    </DetailAssetContentFrame>
  )
}
