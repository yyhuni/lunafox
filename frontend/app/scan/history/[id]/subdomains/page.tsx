import { SubdomainsDetailView } from "@/components/subdomains/subdomains-detail-view"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

export default async function ScanHistorySubdomainsPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params

  return (
    <DetailAssetContentFrame>
      <SubdomainsDetailView scanId={Number(id)} />
    </DetailAssetContentFrame>
  )
}
