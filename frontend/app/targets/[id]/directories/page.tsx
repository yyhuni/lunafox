import { DirectoriesView } from "@/components/directories/directories-view"

export default async function TargetDirectoriesPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const targetId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <DirectoriesView targetId={targetId} />
    </div>
  )
}
