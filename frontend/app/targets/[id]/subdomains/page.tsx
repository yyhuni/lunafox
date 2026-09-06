import { SubdomainsDetailView } from "@/components/subdomains/subdomains-detail-view"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

export default async function TargetSubdomainsPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params

  return (
    <DetailAssetContentFrame>
      <SubdomainsDetailView targetId={Number(id)} />
    </DetailAssetContentFrame>
  )
}
