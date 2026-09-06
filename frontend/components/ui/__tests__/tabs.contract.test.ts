import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/tabs.tsx"), "utf8")
const globals = readFileSync(path.resolve(process.cwd(), "app/globals.css"), "utf8")

function collectSourceFiles(directory: string): string[] {
  return readdirSync(directory).flatMap((entry) => {
    const fullPath = path.join(directory, entry)
    const stats = statSync(fullPath)

    if (stats.isDirectory()) {
      if (entry === "__tests__") return []
      return collectSourceFiles(fullPath)
    }

    return /\.(tsx|ts)$/.test(entry) ? [fullPath] : []
  })
}

describe("tabs contract", () => {
  it("uses Base UI Tabs as the shared primitive baseline", () => {
    expect(source).toContain('from "@base-ui/react/tabs"')
    expect(source).toContain("TabsPrimitive.Root")
    expect(source).toContain("TabsPrimitive.List")
    expect(source).toContain("TabsPrimitive.Tab")
    expect(source).toContain("TabsPrimitive.Panel")
    expect(source).not.toContain("@radix-ui/react-tabs")
  })

  it("hard-cuts Radix-style asChild and forceMount compatibility", () => {
    expect(source).not.toContain("asChild")
    expect(source).not.toContain("forceMount")
    expect(source).not.toContain("render={asChild ? child : render}")
  })

  it("disables native button assertions when a custom render element is provided", () => {
    expect(source).toContain("render,")
    expect(source).toContain("nativeButton,")
    expect(source).toContain("const resolvedNativeButton = nativeButton ?? (render ? false : undefined)")
    expect(source).toContain("nativeButton={resolvedNativeButton}")
    expect(source).not.toContain("nativeButton={false}")
  })

  it("uses Base UI data-active state selectors instead of Radix data-state active selectors", () => {
    expect(source).toContain("data-active:bg-transparent")
    expect(source).toContain("data-active:shadow-none")
    expect(source).toContain("dark:data-active:bg-transparent")
    expect(source).not.toContain("data-[state=active]")
  })

  it("keeps tabs shells token-driven instead of forcing square corners", () => {
    expect(source).toContain('resolvedVariant === "filter" && "radius-surface relative isolate bg-muted text-muted-foreground"')
    expect(source).toContain('resolvedVariant === "filter" && size === "md" && "h-9 p-[3px]"')
    expect(source).toContain('resolvedVariant === "filter" && size === "sm" && "h-8 p-[2px]"')
    expect(source).not.toContain("bg-card text-muted-foreground h-8")
    expect(source).toContain("radius-control-top")
    expect(source).toContain('from "@/lib/typography"')
    expect(source).toContain("textRole.tab")
    expect(source).not.toContain("tracking-[0.12em] uppercase")
    expect(source).toContain("data-active:bg-transparent")
    expect(source).toContain("data-active:shadow-none")
    expect(source).toContain("dark:data-active:bg-transparent")
  })

  it("owns a single sliding active indicator for filter tabs", () => {
    expect(source).toContain('data-slot="tabs-active-indicator"')
    expect(source).toContain('aria-hidden="true"')
    expect(source).toContain("pointer-events-none")
    expect(source).toContain("transition-[transform,width]")
    expect(source).toContain("duration-[var(--motion-duration-fast)]")
    expect(source).toContain("ease-[var(--motion-ease-standard)]")
    expect(source).toContain('resolvedVariant === "filter"')
    expect(source).toContain('data-active:bg-transparent')
    expect(source).toContain("onClickCapture={handleClickCapture}")
    expect(source).toContain("pendingTriggerRef.current = trigger")
  })

  it("keeps a route-tab indicator on the clicked trigger until the committed tab becomes active", () => {
    expect(source).not.toContain("pendingResetRef")
    expect(source).not.toContain("window.setTimeout(() => {")
    expect(source).toContain("const pendingTrigger = pendingTriggerRef.current")
    expect(source).toContain("const committedActiveTrigger = list.querySelector<HTMLElement>('[data-slot=\"tabs-trigger\"][data-active]')")
    expect(source).toContain("committedActiveTrigger === pendingTrigger")
    expect(source).toContain("const activeTrigger = pendingTrigger && list?.contains(pendingTrigger)")
  })

  it("disables filter-tab indicator travel when reduced motion is requested", () => {
    expect(globals).toContain('[data-slot="tabs-active-indicator"]')
    expect(globals).toContain("@media (prefers-reduced-motion: reduce)")
    expect(globals).toContain("transform: translate3d(var(--tabs-active-indicator-x), 0, 0) !important")
  })

  it("keeps minimal tab styling in the component", () => {
    expect(source).toContain("variant === \"minimal-tab\"")
    expect(source).toContain("data-active:border-primary")
    expect(source).toContain("data-active:bg-transparent")
    expect(source).toContain("leading-4")
  })

  it("supports a fixed compact underline for minimal secondary tabs", () => {
    expect(source).toContain('activeIndicator?: "trigger" | "fixed"')
    expect(source).toContain('activeIndicator = "trigger"')
    expect(source).toContain("activeIndicator === \"fixed\"")
    expect(source).toContain("after:w-2")
    expect(source).toContain("after:top-1/2")
    expect(source).toContain("after:translate-y-2")
    expect(source).toContain("data-active:after:bg-primary")
  })

  it("adds semantic tab variants while preserving legacy aliases", () => {
    expect(source).toContain('"filter"')
    expect(source).toContain('"page-nav"')
    expect(source).toContain('"content"')
    expect(source).toContain('"metric"')
    expect(source).toContain('variant === "filter"')
    expect(source).toContain('variant === "page-nav"')
    expect(source).toContain('variant === "content"')
    expect(source).toContain('variant === "metric"')
    expect(source).toContain("inline-flex w-fit max-w-full items-center justify-center")
    expect(source).toContain("h-8 w-full min-w-0 items-end justify-start gap-4 overflow-x-auto overflow-y-hidden bg-transparent border-b border-border p-0")
    expect(source).toContain("-mb-px")
    expect(source).toContain("border-b-[2.5px] border-transparent")
  })

  it("makes content tabs own the underline alignment without page-local guide lines", () => {
    expect(source).toContain('resolvedVariant === "content" && "h-8 w-full min-w-0 items-end')
    expect(source).toContain('resolvedVariant === "content" && "radius-control-top -mb-px')
    expect(source).toContain('resolvedVariant === "content" && "radius-control-top -mb-px text-muted-foreground data-active:bg-transparent data-active:text-foreground data-active:border-primary')
  })

  it("owns multiline metric tabs without an underline rail", () => {
    expect(source).toContain('resolvedVariant === "metric" && "radius-surface grid h-auto w-full grid-cols-3 items-stretch justify-center gap-0 overflow-hidden border border-border/70 bg-transparent p-0"')
    expect(source).toContain('resolvedVariant === "metric" && "h-auto min-h-20 min-w-0 flex-1 flex-col items-stretch justify-center gap-1 rounded-none border-0 border-r border-border/50 bg-transparent px-3 py-3 text-left text-muted-foreground transition-colors hover:bg-muted/30 hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50 data-active:bg-muted/50 data-active:text-foreground data-active:shadow-none last:border-r-0"')
    expect(source).not.toContain('resolvedVariant === "metric" && "-mb-px')
    expect(source).not.toContain('resolvedVariant === "metric" && "border-b-[2.5px]')
    expect(source).not.toContain('resolvedVariant === "metric" && "data-active:border-primary')
  })

  it("owns count badge geometry for tab labels", () => {
    expect(source).toContain("function TabsCountBadge")
    expect(source).toContain('data-slot="tabs-count-badge"')
    expect(source).toContain("radius-pill")
    expect(source).toContain("border-transparent bg-foreground/10")
    expect(source).toContain("tabular-nums")
    expect(source).toContain("TabsCountBadge")
  })

  it("hard-cuts tab callers to Base UI data-active and data-hidden state attributes", () => {
    const files = collectSourceFiles(path.resolve(process.cwd(), "components"))
    const legacyCallers = files.flatMap((filePath) => {
      const fileSource = readFileSync(filePath, "utf8")
      return fileSource.includes("data-[state=active]") || fileSource.includes("data-[state=inactive]")
        ? [path.relative(process.cwd(), filePath)]
        : []
    })

    expect(legacyCallers).toEqual([])
  })
})
