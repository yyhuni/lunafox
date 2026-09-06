import { TargetSettings } from "@/components/target/target-settings"

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
    <div className="flex min-h-0 flex-1 flex-col px-4 lg:px-6">
      <TargetSettings targetId={targetId} section="blacklist" />
    </div>
  )
}
