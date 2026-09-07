import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/response-evidence/response-evidence-panel.tsx"), "utf8")

describe("response-evidence-panel contract", () => {
  it("owns the shared body-first tab and bounded response content behavior", () => {
    expect(source).toContain('import { ScrollArea } from "@/components/ui/scroll-area"')
    expect(source).toContain('import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"')
    expect(source).toContain('defaultValue="body"')
    expect(source).toContain('value="headers"')
    expect(source).toContain('value="location"')
    expect(source).toContain("whitespace-pre-wrap")
    expect(source).toContain("break-words")
    expect(source).toContain("max-h-52")
    expect(source).toContain("emptyLabel")
  })

  it("keeps the shared owner domain-neutral and copy-free", () => {
    expect(source).toContain("ResponseEvidencePanelProps")
    expect(source).toContain("ResponseEvidencePanelLoadingState")
    expect(source).toContain("ResponseEvidencePanelFrame")
    expect(source).toContain("showMetadata")
    expect(source).not.toContain("useTranslations")
    expect(source).not.toContain("@/types/website")
    expect(source).not.toContain("@/types/search")
    expect(source).not.toContain("CopyButton")
  })
})
