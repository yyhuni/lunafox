import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"
import { resolveWebsiteRelationDetailSection } from "@/components/websites/website-relation-detail-sections"
import { WebsiteRelationDetailView } from "@/components/websites/website-relation-detail-view"

export default async function TargetWebsiteDetailPage({
  params,
}: {
  params: Promise<{ id: string; websiteId: string; section?: string[] }>
}) {
  const { id, websiteId, section } = await params

  return (
    <DetailAssetContentFrame>
      <WebsiteRelationDetailView
        targetId={Number(id)}
        websiteId={Number(websiteId)}
        section={resolveWebsiteRelationDetailSection(section?.[0])}
      />
    </DetailAssetContentFrame>
  )
}
