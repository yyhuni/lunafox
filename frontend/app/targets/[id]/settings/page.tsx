import { TargetSettings } from "@/components/target/target-settings"
import { DetailAssetContentFrame } from "@/components/shared/layout/detail-asset-content-frame"

/**
 * Target settings page
 * Opens the target blacklist workspace by default.
 */
export default async function TargetSettingsPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const targetId = Number(id)

  return (
    <DetailAssetContentFrame className="flex min-h-0 flex-1 flex-col">
      <TargetSettings targetId={targetId} section="blacklist" />
    </DetailAssetContentFrame>
  )
}
