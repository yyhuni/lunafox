import { render, screen } from "@testing-library/react"
import { expect, it, vi } from "vitest"
import { RuntimeHeader } from "../scan-runtime-detail-drawer"
import type { ScanRecord } from "@/types/scan.types"

vi.mock("@/hooks/use-scan-workflows", () => ({ useScanWorkflows: () => ({ data: [] }) }))

const scan: ScanRecord = {
  id: 6, targetId: 7, plannedEngineIds: [], triggerType: "manual", inputSource: "scanSnapshot",
  status: "pending", createdAt: "2026-10-20T00:00:00Z", progress: 0,
  assignmentMode: "pinned", agent: "agents/6", agentId: 6, agentName: "worker-06",
  agentStatus: "online", agentDeleted: false, executionBlockedReason: "agent_license_restricted",
}

it("explains license-blocked queued work while preserving the online pinned Agent and clears the warning on renewal", () => {
  const props: Omit<Parameters<typeof RuntimeHeader>[0], "scan"> = {
    tasks: [], now: new Date("2026-10-20T00:00:00Z"), locale: "en",
    t: ((key: string) => key) as Parameters<typeof RuntimeHeader>[0]["t"],
    statusLabel: (key: string) => key,
  }
  const { rerender } = render(<RuntimeHeader scan={scan} {...props} />)
  expect(screen.getByText("runtimeDrawer.licenseRestricted")).toBeInTheDocument()
  expect(screen.getByText("worker-06")).toBeInTheDocument()
  expect(screen.getByText("runtimeDrawer.meta.online")).toBeInTheDocument()
  rerender(<RuntimeHeader scan={{ ...scan, executionBlockedReason: null }} {...props} />)
  expect(screen.queryByText("runtimeDrawer.licenseRestricted")).not.toBeInTheDocument()
  expect(screen.getByText("worker-06")).toBeInTheDocument()
})
