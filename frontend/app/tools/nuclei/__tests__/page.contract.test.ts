import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/tools/nuclei/page.tsx"), "utf8")

describe("page contract", () => {
  it("uses the POC catalog as the route-critical workspace", () => {
    expect(source).toContain("export default async function NucleiPage")
    expect(source).toContain("from \"@/components/tools/nuclei-poc-catalog-page\"")
    expect(source).toContain('from "@/components/common/page-header"')
    expect(source).toContain("<PageHeader")
  })

  it("directly imports the route-critical catalog workspace instead of hiding it behind a null chunk fallback", () => {
    expect(source).not.toContain("lazyPage(")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
    expect(source).not.toContain("NucleiReposPageSkeleton")
    expect(source).not.toContain("PageSectionSkeleton")
  })
})
