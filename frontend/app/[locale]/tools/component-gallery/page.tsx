import { ComponentGallery } from "@/components/demo/component-gallery"
import { getComponentIndex } from "@/components/demo/component-index"

export const runtime = "nodejs"

export default function ComponentGalleryPage() {
  const componentGroups = getComponentIndex()
  return <ComponentGallery componentGroups={componentGroups} />
}
