import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-card-compact.tsx"), "utf8")
const loadingSource = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-card-compact-loading-state.tsx"), "utf8")

describe("agent-card-compact contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function AgentCardCompact")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses shared agent node metric and runtime helpers instead of raw var utilities", () => {
    expect(source).toContain('from "@/components/shared/metrics/segmented-metric-progress"')
    expect(source).toContain("SegmentedMetricProgress")
    expect(source).toContain("getAgentRuntimeStatus")
    expect(source).not.toContain("[var(--success)]")
    expect(source).not.toContain("[var(--warning)]")
    expect(source).not.toContain("[var(--error)]")
  })

  it("uses the shared current-connection display instead of an IP fallback", () => {
    expect(source).toContain("getAgentConnectionIpDisplay")
    expect(source).toContain('t("connectionIp.offline")')
    expect(source).toContain('t("connectionIp.unobserved")')
    expect(source).not.toContain("agentNode.ipAddress")
    expect(source).not.toContain('t("unknownIp")')
  })

  it("uses shared Status component instead of local runtime badge map", () => {
    expect(source).toContain('from "@/components/shared/feedback/status"')
    expect(source).toContain("StatusLabel")
    expect(source).not.toContain("const styles = {")
    expect(source).not.toContain('online: "bg-success/15 text-success border-success/25"')
    expect(source).not.toContain('maintenance: "bg-warning/15 text-warning border-warning/25"')
  })

  it("uses shared runtime shell and heartbeat helpers instead of local status branches", () => {
    expect(source).toContain("getAgentRuntimeShellClass")
    expect(source).toContain("getAgentHeartbeatTextClass")
    expect(source).not.toContain('? "text-error"')
    expect(source).not.toContain('? "text-warning"')
  })

  it("keeps heartbeat and uptime aligned with the task metric columns", () => {
    expect(source).toContain('useFormatHeartbeatTime(7, "compact")')
    expect(source.match(/"grid grid-cols-2 gap-3"/g)).toHaveLength(2)
    // Four cells make up the available-heartbeat metric grid; the missing
    // heartbeat branch also keeps its last-observation row independently aligned.
    expect(source.match(/"flex min-w-0 items-center justify-between gap-2"/g)).toHaveLength(5)
    expect(source).toContain('"min-w-0 truncate text-right font-medium tabular-nums"')
  })

  it("uses the registered task-slot metric icon instead of a raw stable concept icon", () => {
    expect(source).toContain("semanticIcons.metric.taskSlots")
    expect(source).toContain("const TaskSlotsIcon = semanticIcons.metric.taskSlots")
    expect(source).toContain("<TaskSlotsIcon")
    expect(source).not.toContain("IconStack2")
  })

  it("keeps segmented agent node metrics roomy with fewer segments", () => {
    expect(source).toContain("SegmentedMetricProgress")
    expect(source).not.toContain("const segmentCount = 25")
    expect(source).not.toContain("grid-cols-[4.25rem_minmax(0,1fr)_4.75rem]")
  })

  it("keeps running tasks separate from occupied task slots", () => {
    expect(source).toContain('t("metrics.runningTasks")')
    expect(source).toContain("heartbeat.runningTasks")
    expect(source).toContain('t("metrics.usedTaskSlots")')
    expect(source).toContain("heartbeat.taskSlotsUsed")
    expect(source).toContain("agentNode.maxTasks")
  })

  it("does not rely on 9px uppercase labels for readable card metadata", () => {
    expect(source).toContain('from "@/lib/typography"')
    expect(source).toContain("textRole.helperText")
    expect(source).not.toContain("text-[9px]")
  })

  it("routes the compact agent node action menu through the shared dense row owner", () => {
    expect(source).toContain("DenseRowActionMenu")
    expect(source).toContain('triggerSize="icon-sm"')
  })

  it("keeps the compact card loading visual reusable without resolved card dependencies", () => {
    expect(source).toContain('from "./agent-card-compact-loading-state"')
    expect(source).toContain("return <AgentCardCompactLoadingState />")
    expect(loadingSource).toContain("export function AgentCardCompactLoadingState")
    expect(loadingSource).toContain("AGENT_CARD_SHELL_CLASS")
    expect(loadingSource).toContain("AGENT_CARD_HEADER_CLASS")
    expect(loadingSource).toContain("AGENT_CARD_TITLE_CLASS")
    expect(loadingSource).toContain("AGENT_CARD_IP_CLASS")
    expect(loadingSource).toContain("Keep the resolved line box")
    expect(loadingSource).toContain('className={`${AGENT_CARD_TITLE_CLASS} relative`}')
    expect(loadingSource).not.toContain("cn(AGENT_CARD_TITLE_CLASS")
    expect(loadingSource).toContain("!absolute left-0 top-1/2")
    expect(loadingSource).toContain("AGENT_CARD_BODY_CLASS")
    expect(loadingSource).toContain("AGENT_CARD_FOOTER_CLASS")
    expect(source).toContain("AGENT_CARD_TITLE_CLASS")
    expect(source).toContain("AGENT_CARD_IP_CLASS")
    expect(loadingSource).not.toContain("@/hooks/")
    expect(loadingSource).not.toContain("useTranslations")
    expect(loadingSource).toContain('from "@/components/shared/metrics/segmented-metric-progress"')
    expect(loadingSource).toContain("<SegmentedMetricProgress key={metricIndex} loading />")
    expect(loadingSource).not.toContain("DenseRowActionMenu")
  })

  it("uses the shared metric loading variant instead of an independent grid", () => {
    expect(loadingSource).not.toContain("AGENT_RUNTIME_METRIC_GRID_CLASS")
    expect(loadingSource).not.toContain("col-span-2")
    expect(loadingSource).not.toContain("col-span-6")
    expect(loadingSource).not.toContain("ml-auto")
  })
})
