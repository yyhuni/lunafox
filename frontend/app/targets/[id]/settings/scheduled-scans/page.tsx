import { TargetSettings } from "@/components/target/target-settings"

/**
 * Target scheduled scan workspace.
 */
export default async function TargetScheduledScansPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const targetId = Number(id)

  return (
    <div className="flex min-h-0 flex-1 flex-col px-4 lg:px-6">
      <TargetSettings targetId={targetId} section="scheduled-scans" />
    </div>
  )
}
