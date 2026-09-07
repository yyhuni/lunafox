import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import { readdirSync, statSync } from "node:fs"

const source = readFileSync(path.resolve(process.cwd(), "lib/toast-helpers.ts"), "utf8")
const uiFoundationSource = readFileSync(path.resolve(process.cwd(), "components/ui/README.md"), "utf8")
const hooksSource = readFileSync(path.resolve(process.cwd(), "hooks/README.md"), "utf8")

const productionRoots = ["app", "components", "hooks", "lib"]
const directSonnerImportPattern = /from\s+["']sonner["']/
const directSonnerOwners = new Set(["components/ui/sonner.tsx", "lib/toast-helpers.ts"])

function collectProductionFiles(dir: string): string[] {
  const absoluteDir = path.resolve(process.cwd(), dir)
  const files: string[] = []

  for (const entry of readdirSync(absoluteDir)) {
    const absolutePath = path.join(absoluteDir, entry)
    const relativePath = path.relative(process.cwd(), absolutePath)

    if (entry === "__tests__" || entry === "node_modules") continue

    const stat = statSync(absolutePath)
    if (stat.isDirectory()) {
      files.push(...collectProductionFiles(relativePath))
    } else if (/\.(ts|tsx)$/.test(entry)) {
      files.push(relativePath)
    }
  }

  return files
}

describe("toast-helpers contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useToastMessages")
    expect(source).toContain("from \"sonner\"")
  })

  it("owns the production toast feedback API", () => {
    expect(source).toContain("export const toastFeedback")
    expect(source).not.toContain("export const showToast")
  })

  it("documents the standard toast feedback practice", () => {
    expect(uiFoundationSource).toContain("### Toast Standards")
    expect(uiFoundationSource).toContain("Only `frontend/components/ui/sonner.tsx` and `frontend/lib/toast-helpers.ts` may import `sonner` directly.")
    expect(uiFoundationSource).toContain("Production call sites MUST use `useToastMessages` or `toastFeedback`.")
    expect(uiFoundationSource).toContain("limits visible messages to three")
    expect(uiFoundationSource).toContain("The global `Toaster` follows the active LunaFox light/dark theme")
    expect(uiFoundationSource).toContain("One logical async operation MUST use one stable toast id")
    expect(uiFoundationSource).toContain("Terminal feedback MUST replace the loading toast in place with the same id")
    expect(uiFoundationSource).toContain("Collapsed stacking is reserved for distinct operations")
    expect(hooksSource).toContain("`loadingToast.id` is required")
    expect(hooksSource).toContain("Feature hooks MUST NOT call `toast.loading` or `toast.dismiss` manually")
    expect(hooksSource).toContain("Components MUST NOT create terminal feedback for a remote command after `mutateAsync`")
  })

  it("keeps direct sonner imports inside toast owners only", () => {
    const offenders = productionRoots
      .flatMap(collectProductionFiles)
      .filter((filePath) => !directSonnerOwners.has(filePath))
      .filter((filePath) => directSonnerImportPattern.test(readFileSync(path.resolve(process.cwd(), filePath), "utf8")))

    expect(offenders).toEqual([])
  })
})
