import { ScreenshotsGallery } from "@/components/screenshots/screenshots-gallery"

export default async function ScanScreenshotsPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = await params
  const scanId = Number(id)

  return (
    <div className="px-4 lg:px-6">
      <ScreenshotsGallery scanId={scanId} />
    </div>
  )
}
