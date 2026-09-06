import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/server-resource-popover.tsx"), "utf8")

describe("server resource popover contract", () => {
  it("uses the existing overview runtime metrics query", () => {
    expect(source).toContain('from "@/hooks/use-overview"')
    expect(source).toContain("useServerRuntimeMetrics")
    expect(source).not.toContain("getServerRuntimeMetrics")
  })

  it("keeps the header entry global across protected app routes", () => {
    expect(source).toContain("<ServerResourcePopoverContent />")
    expect(source).not.toContain("usePathname")
    expect(source).not.toContain("isOverviewPathname")
    expect(source).not.toContain("/overview/")
  })

  it("does not start runtime polling until the popover is opened", () => {
    expect(source).toContain("const [open, setOpen] = useState(false)")
    expect(source).toContain("<Popover open={open} onOpenChange={setOpen}>")
    expect(source).toContain("{open ? <ServerResourcePopoverContent /> : null}")
  })

  it("uses shared foundation primitives for the trigger, overlay, and segmented metric bars", () => {
    expect(source).toContain('from "@/components/ui/button"')
    expect(source).toContain('from "@/components/ui/popover"')
    expect(source).toContain('from "@/components/shared/metrics/segmented-metric-progress"')
    expect(source).toContain("PopoverTrigger")
    expect(source).toContain("PopoverContent")
    expect(source).toContain("<SegmentedMetricProgress")
  })

  it("uses the shared header placement offset", () => {
    expect(source).toContain('from "@/lib/ui/overlay-styles"')
    expect(source).toContain("shellOverlaySideOffsets.header")
    expect(source).toContain("sideOffset={shellOverlaySideOffsets.header}")
  })

  it("keeps the header title text-only while preserving runtime metric icons", () => {
    expect(source).not.toContain("ServerResourceTitleIcon")
    expect(source).not.toContain("semanticIcons.metric.serverResources")
    expect(source).toContain("IconCpu")
    expect(source).not.toContain("IconServer")
  })

  it("relies on automatic polling instead of a manual refresh action", () => {
    expect(source).toContain("serverResourcesUpdatedAt")
    expect(source).not.toContain("IconRefresh")
    expect(source).not.toContain("serverResourcesRefresh")
    expect(source).not.toContain("refetch()")
  })

  it("does not show agent threshold values in the overview header popover", () => {
    expect(source).not.toContain("threshold:")
    expect(source).not.toContain("threshold={")
    expect(source).not.toContain("/ 85%")
    expect(source).not.toContain("/ 90%")
  })

  it("uses neutral percentage text while retaining status-colored resource bars", () => {
    expect(source).toContain('valueTone="neutral"')
    expect(source).toContain('variant="panel"')
  })
})
