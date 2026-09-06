import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/metric-progress.tsx"), "utf8")

describe("metric-progress contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function MetricProgress")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared agent node metric helpers instead of raw var utilities", () => {
    expect(source).toContain("getAgentMetricBarClass")
    expect(source).toContain("getAgentMetricTextClass")
    expect(source).toContain("getAgentMetricAlertTextClass")
    expect(source).toContain("getAgentMetricTrackClass")
    expect(source).not.toContain("[var(--success)]")
    expect(source).not.toContain("[var(--warning)]")
    expect(source).not.toContain("[var(--error)]")
    expect(source).not.toContain('tone === "error" ? "text-error" : "text-warning"')
    expect(source).not.toContain('status === "critical" && "bg-error/20"')
    expect(source).not.toContain('status === "warning" && "bg-warning/20"')
  })

  it("keeps metric bar animation compositor-friendly", () => {
    expect(source).toContain("indicatorClassName")
    expect(source).not.toContain("transition-[width")
    expect(source).not.toContain("width: `${percentage}%`")
  })
})
