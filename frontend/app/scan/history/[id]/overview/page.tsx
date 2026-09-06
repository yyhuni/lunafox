import { ScanOverview } from "@/components/scan/history/scan-overview"

/**
 * Scan overview page
 * Displays scan statistics and summary information
 */
export default async function ScanOverviewPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const scanId = Number(id)

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col px-4 lg:px-6">
      <ScanOverview scanId={scanId} />
    </div>
  )
}
