import { ScreenshotsGallery } from "@/components/screenshots/screenshots-gallery"

export default async function ScreenshotsPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const targetId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <ScreenshotsGallery targetId={targetId} />
    </div>
  )
}
