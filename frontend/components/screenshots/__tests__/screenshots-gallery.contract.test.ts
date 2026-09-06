import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/screenshots/screenshots-gallery.tsx"), "utf8")

describe("screenshots-gallery contract", () => {
  it("renders gallery failures through the shared app error owner while keeping empty states separate", () => {
    expect(source).toContain("export function ScreenshotsGallery")
    expect(source).toContain('from "@/components/shared/feedback/app-error-state"')
    expect(source).toContain("<AppErrorState")
    expect(source).toContain("<ScreenshotsGalleryEmptyState")
    expect(source).not.toContain("ScreenshotsGalleryErrorState")
  })
})
