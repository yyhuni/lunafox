import { IPAddressesView } from "@/components/ip-addresses/ip-addresses-view"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

export default async function ScanHistoryIPsPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params

  return (
    <DetailAssetContentFrame>
      <IPAddressesView scanId={Number(id)} />
    </DetailAssetContentFrame>
  )
}
