import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/metrics/segmented-metric-progress.tsx"), "utf8")

describe("segmented-metric-progress contract", () => {
  it("owns the shared segmented runtime metric bar", () => {
    expect(source).toContain("export function SegmentedMetricProgress")
    expect(source).toContain("const SEGMENT_COUNT = 18")
    expect(source).toContain('const COMPACT_METRIC_ROW_CLASS = "flex min-w-0 items-center gap-1.5 text-[10px]"')
    expect(source).toContain('const COMPACT_METRIC_LABEL_TRACK_CLASS = "flex w-10 shrink-0 items-center gap-1.5 text-muted-foreground"')
    expect(source).toContain('const COMPACT_METRIC_BAR_TRACK_CLASS = "flex h-1.5 min-w-0 flex-1 gap-0.5"')
    expect(source).toContain('const COMPACT_METRIC_VALUE_TRACK_CLASS = "flex w-12 shrink-0 items-center justify-end gap-1 whitespace-nowrap tabular-nums"')
    expect(source).toContain("/ {thresholdValue}%")
  })

  it("uses the compact metric layout for a native loading variant", () => {
    expect(source).toContain("interface SegmentedMetricProgressLoadingProps")
    expect(source).toContain("loading: true")
    expect(source).toContain("function SegmentedMetricProgressLoadingState")
    expect(source).toContain("if (props.loading)")
    expect(source).toContain("<SegmentedMetricProgressLoadingState")
  })

  it("supports semantic, neutral, and threshold-only tones for segments and values", () => {
    expect(source).toContain('from "@/lib/status-config"')
    expect(source).toContain("getStatusToneBgClass")
    expect(source).toContain("getStatusToneTextClass")
    expect(source).toContain('valueTone?: "status" | "neutral"')
    expect(source).toContain('barTone?: "status" | "neutral" | "threshold"')
    expect(source).toContain('variant?: "compact" | "inline" | "panel" | "runtime-card"')
    expect(source).toContain('barTone = "status"')
    expect(source).toContain('barTone === "neutral"')
    expect(source).toContain('barTone === "threshold"')
    expect(source).toContain("value < threshold")
    expect(source).toContain('"bg-muted-foreground"')
    expect(source).toContain('valueTone = "status"')
    expect(source).toContain('const valueClassName = valueTone === "neutral"')
    expect(source).toContain("? textRole.metadataValueStrong")
    expect(source).toContain("getStatusToneTextClass(getMetricTone(percentage, thresholdValue))")
    expect(source).toContain("className={valueClassName}")
    expect(source).not.toContain("text-emerald")
    expect(source).not.toContain("bg-red")
  })

  it("provides a compact two-tier runtime-card layout for dashboard resource rows", () => {
    expect(source).toContain('variant === "runtime-card"')
    expect(source).toContain('"min-w-0 space-y-1"')
    expect(source).toContain('"flex min-w-0 items-center justify-between gap-2"')
    expect(source).toContain('"flex h-2 min-w-0 flex-1 gap-0.5"')
    expect(source).toContain('"shrink-0 tabular-nums text-muted-foreground"')
  })

  it("supports an inline resource bar for dense comparable rows", () => {
    expect(source).toContain('variant === "inline"')
    expect(source).toContain('"flex min-w-0 flex-1 flex-col gap-1.5"')
    expect(source).toContain('import { Progress } from "@/components/ui/progress"')
    expect(source).toContain('showProgress?: boolean')
    expect(source).toContain('showProgress = true')
    expect(source).toContain('<Progress value={percentage} aria-label={label} className="h-1 bg-muted" indicatorClassName={activeColorClass} />')
    expect(source).toContain("{detail ?? `${percentage.toFixed(0)}%`}")
  })
})
