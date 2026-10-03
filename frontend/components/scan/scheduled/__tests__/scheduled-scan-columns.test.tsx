import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { getStatusToneTextClass } from "@/lib/status-config"
import {
  createScheduledScanColumns,
  type ScheduledScanTranslations,
} from "@/components/scan/scheduled/scheduled-scan-columns"
import type { ScheduledScan } from "@/types/scheduled-scan.types"

const translations: ScheduledScanTranslations = {
  columns: {
    taskName: "Task Name",
    cronExpression: "Schedule",
    scope: "Scope",
    status: "Status",
    nextRun: "Next Run",
    handoffResults: "Handoff Results",
    trigger: "Trigger",
    success: "Success",
    failure: "Failed",
    lastRun: "Last Trigger",
    lastFailure: "Last run failed",
    viewHistory: "View run history",
    viewHistoryHint: "Click to view run history within the retention window",
    hasFailureIndicator: "Has failed runs",
    failureCauses: {
      WORKFLOW_UNAVAILABLE: "referenced scan workflow is unavailable",
      AGENT_NOT_FOUND: "bound agent has been deleted",
      CONFIG_RESOURCE_UNAVAILABLE: "configuration resource is unavailable",
      ENGINE_UNAVAILABLE: "referenced engine is unavailable",
      TARGET_UNAVAILABLE: "scan target is deleted or invalid",
      INTERNAL_UNAVAILABLE: "internal server error",
    },
  },
  actions: {
    editTask: "Edit",
    delete: "Delete",
    openMenu: "Open actions",
    selectAll: "Select all",
    selectRow: "Select row",
  },
  status: {
    enabled: "Enabled",
    disabled: "Disabled",
  },
  cron: {
    everyMinute: "Every minute",
    everyNMinutes: "Every {n} minutes",
    everyHour: "Every hour at {minute}",
    everyNHours: "Every {n} hours at {minute}",
    everyDay: "Every day at {time}",
    everyWeek: "Every {day} at {time}",
    everyMonth: "Every month on {day} at {time}",
    inTimeZone: "{rule} ({timeZone})",
    weekdays: [],
  },
}

const scheduledScan: ScheduledScan = {
  id: 1,
  name: "Daily scan",
  displayName: "Daily scan",
  scanWorkflow: "default",
  organizationId: 1,
  organizationName: "Acme",
  targetId: null,
  targetName: null,
  scanMode: "organization",
  inputSource: "scanSnapshot",
  timeZone: "UTC",
  cronExpression: "0 2 * * *",
  isEnabled: true,
  nextRunTime: null,
  lastRunTime: null,
  runCount: 10,
  successfulHandoffCount: 8,
  failedHandoffCount: 1,
  lastHandoffFailureCause: null, lastHandoffFailureTime: null,
createdAt: "2026-08-09T00:00:00Z",
  updatedAt: "2026-08-09T00:00:00Z",
}

describe("scheduled scan handoff result column", () => {
  it("shows the wall-clock rule with the saved IANA time zone and keeps raw Cron separate", () => {
    const scan = {
      ...scheduledScan,
      timeZone: "Asia/Shanghai",
      cronExpression: "0 19 * * *",
    }
    const column = createScheduledScanColumns({
      formatDate: (value) => value,
      handleEdit: vi.fn(),
      handleDelete: vi.fn(),
      handleToggleStatus: vi.fn(),
      t: translations,
    }).find((item) => (item as { accessorKey?: string }).accessorKey === "cronExpression")

    if (!column || typeof column.cell !== "function") {
      throw new Error("Expected the schedule column cell")
    }

    render(<>{column.cell({ row: { original: scan } } as never)}</>)

    expect(screen.getByText("Every day at 19:00 (Asia/Shanghai)")).toBeVisible()
    expect(screen.getByText("0 19 * * *")).toBeVisible()
  })

  it("renders trigger, successful handoff, and failed handoff counts in one click-through control", () => {
    const column = createScheduledScanColumns({
      formatDate: (value) => value,
      handleEdit: vi.fn(),
      handleDelete: vi.fn(),
      handleToggleStatus: vi.fn(),
      t: translations,
    }).find((item) => item.id === "handoffResults")

    if (!column || typeof column.cell !== "function") {
      throw new Error("Expected the handoff result column cell")
    }

    render(<>{column.cell({ row: { original: scheduledScan } } as never)}</>)

    expect(screen.getByText("Trigger 10")).toBeVisible()
    expect(screen.getByText("Success 8")).toHaveClass(getStatusToneTextClass("success"))
    expect(screen.getByText("Failed 1")).toHaveClass(getStatusToneTextClass("error"))
  })
})

describe("scheduled scan handoff metric opens run history", () => {
  const handoffColumn = () =>
    createScheduledScanColumns({
      formatDate: (value) => value,
      handleEdit: vi.fn(),
      handleDelete: vi.fn(),
      handleToggleStatus: vi.fn(),
      t: translations,
    }).find((item) => item.id === "handoffResults")

  it("activating the metric opens the edit drawer on the occurrences tab", () => {
    const handleEdit = vi.fn()
    const column = createScheduledScanColumns({
      formatDate: (value) => value,
      handleEdit,
      handleDelete: vi.fn(),
      handleToggleStatus: vi.fn(),
      t: translations,
    }).find((item) => item.id === "handoffResults")
    if (!column || typeof column.cell !== "function") {
      throw new Error("Expected the handoff result column cell")
    }
    render(<>{column.cell({ row: { original: scheduledScan } } as never)}</>)

    fireEvent.click(screen.getByRole("button"))

    expect(handleEdit).toHaveBeenCalledWith(scheduledScan, "occurrences")
  })

  it("marks failed schedules with a compact failure indicator and no inline cause line", () => {
    const column = handoffColumn()
    if (!column || typeof column.cell !== "function") {
      throw new Error("Expected the handoff result column cell")
    }
    render(
      <>{column.cell({ row: { original: {
        ...scheduledScan,
        lastHandoffFailureCause: "CONFIG_RESOURCE_UNAVAILABLE",
        lastHandoffFailureTime: "2026-08-09T02:00:00Z",
      } } } as never)}</>
    )
    expect(screen.getByRole("img", { name: "Has failed runs" })).toBeVisible()
    expect(screen.queryByText(/Last run failed/)).not.toBeInTheDocument()
  })

  it("omits the failure indicator once no failed handoffs remain", () => {
    const column = handoffColumn()
    if (!column || typeof column.cell !== "function") {
      throw new Error("Expected the handoff result column cell")
    }
    render(<>{column.cell({ row: { original: { ...scheduledScan, failedHandoffCount: 0 } } } as never)}</>)
    expect(screen.queryByRole("img", { name: "Has failed runs" })).not.toBeInTheDocument()
  })
})
