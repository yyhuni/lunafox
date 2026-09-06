import { VulnerabilitiesDetailView } from "@/components/vulnerabilities/vulnerabilities-detail-view"

export default async function ScanHistoryVulnerabilitiesPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const scanId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <VulnerabilitiesDetailView scanId={scanId} />
    </div>
  )
}
