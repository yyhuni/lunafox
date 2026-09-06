import { DirectoriesView } from "@/components/directories/directories-view"

export default async function ScanDirectoriesPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const scanId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <DirectoriesView scanId={scanId} />
    </div>
  )
}
