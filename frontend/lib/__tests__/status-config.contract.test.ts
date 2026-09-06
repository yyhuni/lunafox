import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/status-config.ts"), "utf8")

describe("status-config contract", () => {
  it("exports shared status ownership helpers", () => {
    expect(source).toContain("export const SCAN_STATUS_CONFIG")
    expect(source).toContain("export const RUNTIME_STATUS_CONFIG")
    expect(source).toContain("export function getScanStatusClasses")
    expect(source).toContain("export function getRuntimeStatusClasses")
    expect(source).toContain("export function getStatusToneColorVar")
    expect(source).toContain("export function getScanStatusMetricTone")
    expect(source).toContain("export function getStatusToneInteractiveOutlineClass")
    expect(source).toContain('export type TrendTone = "positive" | "negative" | "neutral"')
    expect(source).toContain("export function getTrendToneTextClass")
    expect(source).toContain("export function getTrendToneColorVar")
    expect(source).toContain('positive: "text-trend-positive"')
    expect(source).toContain('negative: "text-trend-negative"')
    expect(source).toContain('neutral: "text-trend-neutral"')
  })

  it("keeps scan status metric tones derived from the shared scan status config", () => {
    expect(source).toContain("type StatusMetricToneClassNames")
    expect(source).toContain("const config = getScanStatusConfig(status)")
    expect(source).toContain("const textClassName = getStatusToneTextClass(config.tone)")
    expect(source).toContain("return { label: textClassName, dot: getStatusToneBgClass(config.tone), footer: textClassName }")
  })

  it("keeps muted lifecycle badges visibly outlined on neutral surfaces", () => {
    expect(source).toContain('muted: "border-border bg-muted/10 text-muted-foreground"')
    expect(source).not.toContain('muted: "border-muted/20 bg-muted/10 text-muted-foreground"')
  })
})
