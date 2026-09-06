import { render, screen } from "@testing-library/react"
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
  cronExpression: "0 2 * * *",
  isEnabled: true,
  nextRunTime: null,
  lastRunTime: null,
  runCount: 10,
  successfulHandoffCount: 8,
  failedHandoffCount: 1,
  createdAt: "2026-08-09T00:00:00Z",
  updatedAt: "2026-08-09T00:00:00Z",
}

describe("scheduled scan handoff result column", () => {
  it("renders trigger, successful handoff, and failed handoff counts without a command surface", () => {
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
    expect(screen.queryByRole("button")).not.toBeInTheDocument()
  })
})
