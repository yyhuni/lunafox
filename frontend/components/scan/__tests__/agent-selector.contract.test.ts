import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/agent-selector.tsx"), "utf8")
const hookSource = readFileSync(path.resolve(process.cwd(), "hooks/use-scan-agent-picker.ts"), "utf8")
const initiateSectionsSource = readFileSync(
  path.resolve(process.cwd(), "components/scan/initiate-scan-dialog-sections.tsx"),
  "utf8",
)
const initiateSource = readFileSync(path.resolve(process.cwd(), "components/scan/initiate-scan-dialog.tsx"), "utf8")
const quickSource = readFileSync(path.resolve(process.cwd(), "components/scan/quick-scan-dialog.tsx"), "utf8")
const scheduledCreateSource = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/create-scheduled-scan-dialog.tsx"), "utf8")
const scheduledEditSource = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/edit-scheduled-scan-dialog.tsx"), "utf8")

describe("scan agent selector contract", () => {
  it("uses a bounded searchable overlay instead of expanding the scan form", () => {
    expect(source).toContain("<ScanSearchablePicker")
    expect(source).toContain("searchPlaceholder")
    expect(source).toContain("searchEmpty")
    expect(source).toContain('from "./scan-searchable-picker"')
    expect(source).not.toContain("<Select")
  })

  it("uses bounded server search and incremental pages while preserving live resource signals", () => {
    expect(source).toContain("useScanAgentPicker")
    expect(hookSource).toContain("SCAN_AGENT_PICKER_PAGE_SIZE = 50")
    expect(hookSource).toContain("SCAN_AGENT_PICKER_SEARCH_DEBOUNCE_MS = 250")
    expect(hookSource).toContain('SCAN_AGENT_PICKER_ORDER_BY = "createdAt desc"')
    expect(hookSource).toContain("nextPageToken")
    expect(hookSource).toContain("agent.resourceName")
    expect(source).toContain("IntersectionObserver")
    expect(source).toContain("picker.retryNextPage()")
    expect(source).not.toContain("pageSize: 1000")
    expect(source).not.toContain("useAgents(")
    expect(source).toContain("agent.observedHostname")
    expect(source).toContain("agent.connectionIp")
    expect(source).toContain("heartbeat.cpu")
    expect(source).toContain("heartbeat.mem")
    expect(source).toContain("heartbeat.disk")
    expect(source).toContain("heartbeat.taskSlotsUsed")
    expect(source).toContain("SegmentedMetricProgress")
    expect(source).toContain('semanticIcons.metric.taskSlots')
    expect(source).toContain('label={tAgents("metrics.usedTaskSlots")}')
    expect(source).toContain('detail={`${heartbeat.taskSlotsUsed}/${agent.maxTasks}`}')
    expect(source).not.toContain("showProgress={false}")
    expect(source).not.toContain("mayWait")
  })

  it("keeps unhealthy and offline Agents selectable", () => {
    expect(source).toContain("onSelect={() => onChange(agent.id)}")
    expect(source).not.toContain("disabled={warning}")
  })

  it("keeps offline Agents on the shared metric grid with neutral placeholders", () => {
    expect(source).toContain('agent.status === "offline"')
    expect(source).toContain('detail="—"')
    expect(source).toContain('detail="0/0"')
    expect(source).toContain('valueTone="neutral"')
    expect(source).toContain('barTone="neutral"')
  })

  it("defers the Agent query until the picker opens and keeps selected state on its trigger", () => {
    expect(source).toContain('const [isPickerOpen, setIsPickerOpen] = React.useState(false)')
    expect(source).toContain('open: isPickerOpen')
    expect(source).toContain('onOpenChange={handlePickerOpenChange}')
    expect(hookSource).toContain("enabled: open")
    expect(source).not.toContain('<Check')
  })

  it("restores selected identity through canonical detail without collection inference", () => {
    expect(source).toContain("useSelectedAgentDetail(selectedResourceName)")
    expect(source).toContain("agentName(value)")
    expect(source).toContain('t("unavailableHint")')
    expect(source).not.toContain("agents.find")
  })

  it("shares the selector across normal, batch, quick, scheduled-create, and scheduled-edit flows", () => {
    expect(initiateSource).toContain("<InitiateScanExecutionOptions")
    expect(initiateSource).toContain("export function BulkInitiateScanDrawer")
    expect(initiateSource).toContain("<InitiateScanDrawer")
    expect(initiateSectionsSource).toContain("<ScanAgentSelector")
    expect(quickSource).toContain("<InitiateScanExecutionOptions")
    expect(scheduledCreateSource).toContain("<ScanAgentSelector")
    expect(scheduledEditSource).toContain("<InitiateScanExecutionOptions")
    expect(scheduledEditSource).not.toContain("<ScanAgentSelector")
  })

  it("marks automatic assignment as the recommended option", () => {
    expect(source).toContain('value="automatic"')
    expect(source).toContain("semanticIcons.concept.automaticAssignment")
    expect(source).toContain('<span className="min-w-0 flex-1">')
    expect(source).toContain('<Badge variant="success" className="shrink-0">{t("recommended")}</Badge>')
    expect(source).not.toContain('{value === null ? <Check className="size-4 shrink-0 text-primary" /> : null}')
  })

  it("keeps identity metadata aligned without hiding the full node name from assistive hover", () => {
    expect(source).toContain('items-center justify-between gap-2 sm:w-36')
    expect(source).toContain('min-w-0 flex-1 truncate')
    expect(source).toContain('title={agent.displayName || agent.name}')
  })
})
