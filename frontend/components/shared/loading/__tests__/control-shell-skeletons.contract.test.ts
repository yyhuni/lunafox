import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const searchToolbarSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/loading/search-toolbar-skeleton.tsx"),
  "utf8"
)
const selectShellSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/loading/select-shell-skeleton.tsx"),
  "utf8"
)
const compactPaginationSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/loading/compact-pagination-skeleton.tsx"),
  "utf8"
)
const pageSectionSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/loading/page-section-skeleton.tsx"),
  "utf8"
)
const settingsPageSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/loading/settings-page-skeleton.tsx"),
  "utf8"
)
const interactionLoadingDialogSource = readFileSync(
  path.resolve(process.cwd(), "components/shared/loading/interaction-loading-dialog.tsx"),
  "utf8"
)

describe("control-shell skeleton contracts", () => {
  it("keeps search toolbar skeletons on shared input geometry and shared action placeholders", () => {
    expect(searchToolbarSource).toContain("export function SearchToolbarSkeleton")
    expect(searchToolbarSource).toContain('data-slot="search-toolbar-skeleton"')
    expect(searchToolbarSource).toContain('from "@/components/ui/input"')
    expect(searchToolbarSource).toContain('from "@/components/shared/loading/action-skeleton"')
    expect(searchToolbarSource).toContain('from "@/types/data-table.types"')
    expect(searchToolbarSource).toContain("leadingSpaceClassName?: string")
    expect(searchToolbarSource).toContain('inputWidthMode?: "fixed" | "fill"')
    expect(searchToolbarSource).toContain("showButton = false")
    expect(searchToolbarSource).toContain('inputWidthMode = "fixed"')
    expect(searchToolbarSource).toContain('const defaultInputWidthClassName = "w-full sm:w-72 lg:w-80"')
    expect(searchToolbarSource).toContain('const fillInputWidthClassName = "w-full"')
    expect(searchToolbarSource).toContain('const resolvedInputWidthClassName = inputWidthMode === "fill" ? fillInputWidthClassName : defaultInputWidthClassName')
    expect(searchToolbarSource).toContain('const resolvedLeadingSpaceClassName = leadingSpaceClassName ?? (!showButton ? "size-4" : undefined)')
    expect(searchToolbarSource).toContain('className={cn("shrink-0 opacity-0", resolvedLeadingSpaceClassName)}')
    expect(searchToolbarSource).not.toContain("showInlineIcon")
    expect(searchToolbarSource).not.toContain("sm:w-44")
    expect(searchToolbarSource).not.toContain("size-4 rounded-full")
    expect(searchToolbarSource).not.toContain("border border-input bg-background")
  })

  it("keeps select shell skeletons on the shared select trigger contract", () => {
    expect(selectShellSource).toContain("export function SelectShellSkeleton")
    expect(selectShellSource).toContain('data-slot="select-shell-skeleton"')
    expect(selectShellSource).toContain('from "@/components/ui/select"')
    expect(selectShellSource).toContain("<SelectTrigger")
    expect(selectShellSource).toContain("disabled:opacity-100")
    expect(selectShellSource).toContain("leadingSpaceClassName?: string")
    expect(selectShellSource).toContain('className={cn("shrink-0 opacity-0", leadingSpaceClassName)}')
    expect(selectShellSource).not.toContain("showLeadingIconPlaceholder")
    expect(selectShellSource).not.toContain('className="size-4 rounded-full"')
  })

  it("keeps compact pagination skeletons on shared select and action owners", () => {
    expect(compactPaginationSource).toContain("export function CompactPaginationSkeleton")
    expect(compactPaginationSource).toContain('data-slot="compact-pagination-skeleton"')
    expect(compactPaginationSource).toContain('from "@/components/shared/loading/select-shell-skeleton"')
    expect(compactPaginationSource).toContain('from "@/components/shared/loading/action-skeleton"')
    expect(compactPaginationSource).toContain('size="icon-sm"')
    expect(compactPaginationSource).toContain('mode?: "numbered" | "cursor"')
    expect(compactPaginationSource).toContain('mode = "numbered"')
    expect(compactPaginationSource).toContain("showSummary?: boolean")
    expect(compactPaginationSource).toContain("buttonCount?: 2 | 3 | 4")
    expect(compactPaginationSource).toContain("sm:gap-4")
  })

  it("keeps generic page-section toolbar search on the shared search toolbar skeleton", () => {
    expect(pageSectionSource).toContain('from "@/components/shared/loading/search-toolbar-skeleton"')
    expect(pageSectionSource).toContain("<SearchToolbarSkeleton")
    expect(pageSectionSource).toContain('inputWidthMode="fill"')
    expect(pageSectionSource).toContain('toolbarDensity="standard"')
    expect(pageSectionSource).not.toContain('Skeleton className="h-9 w-full max-w-sm rounded-lg"')
  })

  it("keeps generic settings input placeholders on the shared input shell", () => {
    expect(settingsPageSource).toContain('from "@/components/ui/input"')
    expect(settingsPageSource).toContain('<Input aria-hidden="true" disabled tabIndex={-1} className="disabled:opacity-100" />')
    expect(settingsPageSource).not.toContain('Skeleton className="h-10 w-full rounded-lg"')
  })

  it("keeps interaction dialog form and action placeholders on shared control shells", () => {
    expect(interactionLoadingDialogSource).toContain('from "@/components/ui/input"')
    expect(interactionLoadingDialogSource).toContain('from "@/components/shared/loading/action-skeleton"')
    expect(interactionLoadingDialogSource).toContain('<Input aria-hidden="true" disabled tabIndex={-1} className="disabled:opacity-100" />')
    expect(interactionLoadingDialogSource).toContain('<ActionSkeleton widthClassName="w-20" />')
    expect(interactionLoadingDialogSource).toContain('<ActionSkeleton widthClassName="w-28" emphasis="primary" />')
    expect(interactionLoadingDialogSource).not.toContain('Skeleton className="h-10 w-full rounded-md"')
    expect(interactionLoadingDialogSource).not.toContain('Skeleton className="h-9 w-20 rounded-md"')
    expect(interactionLoadingDialogSource).not.toContain('Skeleton className="h-9 w-28 rounded-md"')
  })
})
